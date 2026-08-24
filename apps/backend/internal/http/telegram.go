package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type telegramService struct {
	server  *Server
	client  *http.Client
	apiBase string
	kick    chan struct{}
}

type telegramUser struct {
	Username string `json:"username"`
}

type telegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

type telegramResponse[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      T      `json:"result"`
}

var telegramTokenPattern = regexp.MustCompile(`^\d{5,}:[A-Za-z0-9_-]{30,}$`)
var trackingCodePattern = regexp.MustCompile(`(?i)\b[23456789A-HJ-NP-Z]{10}\b`)

func newTelegramService(server *Server) *telegramService {
	return &telegramService{server: server, client: &http.Client{Timeout: 40 * time.Second}, apiBase: "https://api.telegram.org", kick: make(chan struct{}, 1)}
}

func (s *Server) StartTelegramBot(ctx context.Context) {
	go s.telegram.run(ctx)
}

func (service *telegramService) notifyConfigurationChanged() {
	select {
	case service.kick <- struct{}{}:
	default:
	}
}

func (service *telegramService) run(ctx context.Context) {
	var offset int64
	for {
		token, enabled, err := service.configuredToken(ctx)
		if err != nil || !enabled || token == "" {
			select {
			case <-ctx.Done():
				return
			case <-service.kick:
				offset = 0
			case <-time.After(20 * time.Second):
			}
			continue
		}
		updates, err := service.getUpdates(ctx, token, offset)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-service.kick:
				offset = 0
			case <-time.After(5 * time.Second):
			}
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message != nil {
				service.handleMessage(ctx, token, update.Message.Chat.ID, update.Message.Text)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-service.kick:
			offset = 0
		default:
		}
	}
}

func (service *telegramService) configuredToken(ctx context.Context) (string, bool, error) {
	var encrypted []byte
	var enabled bool
	if err := service.server.db.QueryRow(ctx, `SELECT telegram_bot_token,telegram_bot_enabled FROM settings WHERE id=true`).Scan(&encrypted, &enabled); err != nil {
		return "", false, err
	}
	if len(encrypted) == 0 {
		return "", enabled, nil
	}
	token, err := service.server.decryptTelegramToken(encrypted)
	return token, enabled, err
}

func (service *telegramService) telegramURL(token, method string) string {
	return service.apiBase + "/bot" + token + "/" + method
}

func (service *telegramService) getMe(ctx context.Context, token string) (telegramUser, error) {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, service.telegramURL(token, "getMe"), nil)
	response, err := service.client.Do(request)
	if err != nil {
		return telegramUser{}, err
	}
	defer response.Body.Close()
	var result telegramResponse[telegramUser]
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return telegramUser{}, err
	}
	if !result.OK {
		return telegramUser{}, errors.New(result.Description)
	}
	return result.Result, nil
}

func (service *telegramService) getUpdates(ctx context.Context, token string, offset int64) ([]telegramUpdate, error) {
	values := url.Values{"timeout": {"25"}, "allowed_updates": {`["message"]`}}
	if offset > 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, service.telegramURL(token, "getUpdates")+"?"+values.Encode(), nil)
	response, err := service.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var result telegramResponse[[]telegramUpdate]
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, errors.New(result.Description)
	}
	return result.Result, nil
}

func extractTrackingCode(text string) string {
	if match := trackingCodePattern.FindString(strings.ToUpper(text)); match != "" {
		return match
	}
	return ""
}

func (service *telegramService) handleMessage(ctx context.Context, token string, chatID int64, text string) {
	code := extractTrackingCode(text)
	if code == "" {
		_ = service.sendMessage(ctx, token, chatID, "Отправьте 10-значный код заказа. Он указан в карточке заказа PrintForge.")
		return
	}
	order, err := service.server.loadTrackedOrder(ctx, code)
	if err != nil {
		_ = service.sendMessage(ctx, token, chatID, "Заказ с таким кодом не найден. Проверьте код и попробуйте ещё раз.")
		return
	}
	message := trackingMessage(order)
	if base := strings.TrimRight(order.PublicBaseURL, "/"); base != "" {
		message += "\n\n<a href=\"" + html.EscapeString(base+"/track/"+order.TrackingCode) + "\">Открыть красивую страницу заказа</a>"
	}
	_ = service.sendMessage(ctx, token, chatID, message)
	for index, model := range order.Models {
		if index >= 5 {
			break
		}
		if model.PreviewPath != nil {
			previewPath := filepath.Join(service.server.cfg.UploadDir, "models", filepath.Base(*model.PreviewPath))
			_ = service.sendFile(ctx, token, "sendPhoto", "photo", chatID, previewPath, "Модель: "+model.Name)
		}
		var storage, filename string
		if err := service.server.db.QueryRow(ctx, `SELECT storage_path,original_filename FROM models WHERE id=$1`, model.ID).Scan(&storage, &filename); err == nil {
			modelPath := filepath.Join(service.server.cfg.UploadDir, "models", filepath.Base(storage))
			_ = service.sendFile(ctx, token, "sendDocument", "document", chatID, modelPath, "Скачать "+filename)
		}
	}
}

func (service *telegramService) sendMessage(ctx context.Context, token string, chatID int64, text string) error {
	values := url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "text": {text}, "parse_mode": {"HTML"}, "disable_web_page_preview": {"true"}}
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, service.telegramURL(token, "sendMessage"), strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := service.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("telegram sendMessage returned %s", response.Status)
	}
	return nil
}

func (service *telegramService) sendFile(ctx context.Context, token, method, field string, chatID int64, filename, caption string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	_ = writer.WriteField("caption", caption)
	part, err := writer.CreateFormFile(field, filepath.Base(filename))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, service.telegramURL(token, method), &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := service.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("telegram %s returned %s", method, response.Status)
	}
	return nil
}

func (s *Server) encryptTelegramToken(token string) ([]byte, error) {
	key := sha256.Sum256([]byte(s.cfg.JWTSecret + ":telegram"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return append(nonce, gcm.Seal(nil, nonce, []byte(token), nil)...), nil
}

func (s *Server) decryptTelegramToken(encrypted []byte) (string, error) {
	key := sha256.Sum256([]byte(s.cfg.JWTSecret + ":telegram"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(encrypted) < gcm.NonceSize() {
		return "", errors.New("telegram token is corrupted")
	}
	plain, err := gcm.Open(nil, encrypted[:gcm.NonceSize()], encrypted[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Server) updateTelegramSettings(w http.ResponseWriter, r *http.Request) {
	if currentUser(r).Role != "ADMIN" {
		writeJSON(w, 403, apiError{Error: "administrator access required"})
		return
	}
	var input struct {
		Token         string `json:"token"`
		Enabled       bool   `json:"enabled"`
		RemoveToken   bool   `json:"removeToken"`
		PublicBaseURL string `json:"publicBaseUrl"`
	}
	if decodeJSON(r, &input) != nil {
		badRequest(w, "invalid telegram settings")
		return
	}
	input.PublicBaseURL = strings.TrimRight(strings.TrimSpace(input.PublicBaseURL), "/")
	if input.PublicBaseURL == "" || (!strings.HasPrefix(input.PublicBaseURL, "http://") && !strings.HasPrefix(input.PublicBaseURL, "https://")) {
		badRequest(w, "publicBaseUrl must be an HTTP or HTTPS address")
		return
	}
	if input.RemoveToken {
		_, err := s.db.Exec(r.Context(), `UPDATE settings SET public_base_url=$1,telegram_bot_token=NULL,telegram_bot_username=NULL,telegram_bot_enabled=false,updated_at=now() WHERE id=true`, input.PublicBaseURL)
		if err != nil {
			writeJSON(w, 500, apiError{Error: "could not remove telegram bot"})
			return
		}
		s.telegram.notifyConfigurationChanged()
		s.getSettings(w, r)
		return
	}
	if strings.TrimSpace(input.Token) != "" {
		token := strings.TrimSpace(input.Token)
		if !telegramTokenPattern.MatchString(token) {
			badRequest(w, "telegram bot token has an invalid format")
			return
		}
		bot, err := s.telegram.getMe(r.Context(), token)
		if err != nil {
			badRequest(w, "Telegram did not accept this bot token")
			return
		}
		encrypted, err := s.encryptTelegramToken(token)
		if err != nil {
			writeJSON(w, 500, apiError{Error: "could not protect telegram token"})
			return
		}
		_, err = s.db.Exec(r.Context(), `UPDATE settings SET public_base_url=$1,telegram_bot_token=$2,telegram_bot_username=$3,telegram_bot_enabled=$4,updated_at=now() WHERE id=true`, input.PublicBaseURL, encrypted, bot.Username, input.Enabled)
		if err != nil {
			writeJSON(w, 500, apiError{Error: "could not save telegram settings"})
			return
		}
	} else {
		var configured bool
		_ = s.db.QueryRow(r.Context(), `SELECT telegram_bot_token IS NOT NULL FROM settings WHERE id=true`).Scan(&configured)
		if input.Enabled && !configured {
			badRequest(w, "enter a bot token before enabling Telegram")
			return
		}
		_, err := s.db.Exec(r.Context(), `UPDATE settings SET public_base_url=$1,telegram_bot_enabled=$2,updated_at=now() WHERE id=true`, input.PublicBaseURL, input.Enabled)
		if err != nil {
			writeJSON(w, 500, apiError{Error: "could not save telegram settings"})
			return
		}
	}
	s.telegram.notifyConfigurationChanged()
	s.getSettings(w, r)
}
