package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/printforge/apps/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type readSeekCloser interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

type objectStorage interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Open(context.Context, string) (readSeekCloser, int64, time.Time, error)
	Remove(context.Context, string) error
	LocalPath(string) (string, bool)
	Driver() string
}

type localObjectStorage struct {
	root string
}

func cleanObjectKey(key string) (string, error) {
	cleaned := filepath.ToSlash(filepath.Clean("/" + strings.TrimSpace(key)))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid object key")
	}
	return cleaned, nil
}

func (store *localObjectStorage) path(key string) (string, error) {
	cleaned, err := cleanObjectKey(key)
	if err != nil {
		return "", err
	}
	return filepath.Join(store.root, filepath.FromSlash(cleaned)), nil
}

func (store *localObjectStorage) Put(_ context.Context, key string, reader io.Reader, size int64, _ string) error {
	path, err := store.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(reader, size+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written != size {
		_ = os.Remove(path)
		if copyErr == nil && closeErr == nil {
			return fmt.Errorf("store local object: wrote %d bytes, expected %d", written, size)
		}
		return fmt.Errorf("store local object: %w", errorsJoin(copyErr, closeErr))
	}
	return nil
}

func (store *localObjectStorage) Open(_ context.Context, key string) (readSeekCloser, int64, time.Time, error) {
	path, err := store.path(key)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) && !strings.Contains(filepath.ToSlash(key), "/") {
		path = filepath.Join(store.root, "models", filepath.Base(key))
		file, err = os.Open(path)
	}
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, time.Time{}, err
	}
	return file, info.Size(), info.ModTime(), nil
}

func (store *localObjectStorage) Remove(_ context.Context, key string) error {
	path, err := store.path(key)
	if err != nil {
		return err
	}
	removeErr := os.Remove(path)
	if os.IsNotExist(removeErr) && !strings.Contains(filepath.ToSlash(key), "/") {
		removeErr = os.Remove(filepath.Join(store.root, "models", filepath.Base(key)))
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return nil
}

func (store *localObjectStorage) LocalPath(key string) (string, bool) {
	path, err := store.path(key)
	if err == nil {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) && !strings.Contains(filepath.ToSlash(key), "/") {
			path = filepath.Join(store.root, "models", filepath.Base(key))
		}
	}
	return path, err == nil
}

func (store *localObjectStorage) Driver() string { return "local" }

type s3ObjectStorage struct {
	client *minio.Client
	bucket string
}

func (store *s3ObjectStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	cleaned, err := cleanObjectKey(key)
	if err != nil {
		return err
	}
	_, err = store.client.PutObject(ctx, store.bucket, cleaned, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (store *s3ObjectStorage) Open(ctx context.Context, key string) (readSeekCloser, int64, time.Time, error) {
	cleaned, err := cleanObjectKey(key)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	object, err := store.client.GetObject(ctx, store.bucket, cleaned, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	info, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, 0, time.Time{}, err
	}
	return object, info.Size, info.LastModified, nil
}

func (store *s3ObjectStorage) Remove(ctx context.Context, key string) error {
	cleaned, err := cleanObjectKey(key)
	if err != nil {
		return err
	}
	return store.client.RemoveObject(ctx, store.bucket, cleaned, minio.RemoveObjectOptions{})
}

func (store *s3ObjectStorage) LocalPath(string) (string, bool) { return "", false }
func (store *s3ObjectStorage) Driver() string                  { return "s3" }

func newObjectStorage(cfg config.Config) (objectStorage, error) {
	if cfg.StorageDriver != "s3" {
		return &localObjectStorage{root: cfg.UploadDir}, nil
	}
	endpoint := cfg.S3Endpoint
	secure := cfg.S3UseSSL
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		endpoint = parsed.Host
		secure = parsed.Scheme != "http"
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure:       secure,
		Region:       cfg.S3Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, err
	}
	return &s3ObjectStorage{client: client, bucket: cfg.S3Bucket}, nil
}

func modelObjectKey(storage string) string {
	if strings.Contains(storage, "/") {
		return storage
	}
	return "models/" + filepath.Base(storage)
}

func errorsJoin(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
