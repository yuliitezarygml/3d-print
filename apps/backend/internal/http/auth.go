package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/local/printforge/apps/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	cfg config.Config
	db  *pgxpool.Pool
}

type claims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	Type string `json:"type"`
	jwt.RegisteredClaims
}

type tokenPair struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresIn    int64    `json:"expiresIn"`
	User         authUser `json:"user"`
}

func newAuthService(cfg config.Config, db *pgxpool.Pool) *authService {
	return &authService{cfg: cfg, db: db}
}

func (a *authService) issueTokens(ctx context.Context, user authUser) (tokenPair, error) {
	now := time.Now()
	accessClaims := claims{
		Name: user.Name, Role: user.Role, Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.AccessTokenTTL))},
	}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		return tokenPair{}, err
	}
	refreshID := uuid.NewString()
	refreshClaims := claims{
		Name: user.Name, Role: user.Role, Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{ID: refreshID, Subject: user.ID, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.RefreshTokenTTL))},
	}
	refresh, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		return tokenPair{}, err
	}
	if _, err := a.db.Exec(ctx, `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`, refreshID, user.ID, tokenHash(refresh), now.Add(a.cfg.RefreshTokenTTL)); err != nil {
		return tokenPair{}, err
	}
	return tokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(a.cfg.AccessTokenTTL.Seconds()), User: user}, nil
}

func (a *authService) parseToken(raw, expectedType string) (*claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	value, ok := parsed.Claims.(*claims)
	if !ok || value.Type != expectedType {
		return nil, errors.New("invalid token type")
	}
	return value, nil
}

func (a *authService) verifyAccessToken(raw string) (authUser, error) {
	value, err := a.parseToken(raw, "access")
	if err != nil {
		return authUser{}, err
	}
	return authUser{ID: value.Subject, Name: value.Name, Role: value.Role}, nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil || input.Email == "" || input.Password == "" {
		badRequest(w, "email and password are required")
		return
	}
	var user authUser
	var hash string
	err := s.db.QueryRow(r.Context(), `SELECT id, name, role, password_hash FROM users WHERE lower(email) = lower($1)`, input.Email).Scan(&user.ID, &user.Name, &user.Role, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid email or password"})
		return
	}
	pair, err := s.auth.issueTokens(r.Context(), user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "could not create session"})
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := decodeJSON(r, &input); err != nil || input.RefreshToken == "" {
		badRequest(w, "refreshToken is required")
		return
	}
	value, err := s.auth.parseToken(input.RefreshToken, "refresh")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid refresh token"})
		return
	}
	var user authUser
	err = s.db.QueryRow(r.Context(), `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id`, value.ID, tokenHash(input.RefreshToken)).Scan(&user.ID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "refresh token already used or expired"})
		return
	}
	err = s.db.QueryRow(r.Context(), `SELECT name, role FROM users WHERE id = $1`, user.ID).Scan(&user.Name, &user.Role)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "user not found"})
		return
	}
	pair, err := s.auth.issueTokens(r.Context(), user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "could not refresh session"})
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refreshToken"`
	}
	if decodeJSON(r, &input) == nil && input.RefreshToken != "" {
		_, _ = s.db.Exec(r.Context(), `UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash(input.RefreshToken))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

var _ = pgx.ErrNoRows
