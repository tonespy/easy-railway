package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/argon2"
)

// magicHeader is the prefix written to encrypted credential files.
var magicHeader = []byte("ERENC")

const (
	saltLen  = 16
	nonceLen = 12

	// Argon2id parameters tuned for interactive CLI latency.
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MiB.
	argonThreads = 4
	argonKeyLen  = 32 // 256-bit AES key.
)

// Token type constants stored in Credentials.Type.
const (
	TypeAccount   = "account"
	TypeWorkspace = "workspace"
	TypeProject   = "project"
)

// Credentials holds the persisted authentication state.
type Credentials struct {
	Type         string    `json:"type"`                    // "account", "workspace", or "project".
	Token        string    `json:"token,omitempty"`         // API key.
	AccessToken  string    `json:"access_token,omitempty"`  // OAuth access token.
	RefreshToken string    `json:"refresh_token,omitempty"` // OAuth refresh token.
	ExpiresAt    time.Time `json:"expires_at,omitempty"`    // OAuth token expiry.

	// Account metadata (from "me" query).
	UserID    string `json:"user_id,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`

	// Workspace metadata (from "workspace" query).
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspaceName string `json:"workspace_name,omitempty"`

	// Project metadata (from "projectToken" query).
	ProjectTokenID   string `json:"project_token_id,omitempty"`
	ProjectTokenName string `json:"project_token_name,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	EnvironmentID    string `json:"environment_id,omitempty"`
}

// BearerToken returns the token to use in Authorization headers.
// It prefers AccessToken (OAuth) over Token (API key).
func (c *Credentials) BearerToken() string {
	if c.AccessToken != "" {
		return c.AccessToken
	}

	return c.Token
}

// IsExpired reports whether the credential's access token has expired.
// Returns false for API keys (ExpiresAt is zero).
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}

	return time.Now().After(c.ExpiresAt)
}

// NeedsRefresh reports whether the credential is expired and has a refresh token available.
func (c *Credentials) NeedsRefresh() bool {
	return c.IsExpired() && c.RefreshToken != ""
}

// Store persists and retrieves Credentials.
type Store interface {
	// Load reads credentials. Returns (nil, nil) if no credentials are stored.
	Load() (*Credentials, error)
	// Save writes credentials atomically.
	Save(c *Credentials) error
	// Delete removes stored credentials.
	Delete() error
}

// FileStore stores credentials as a plain JSON file with 0600 permissions.
type FileStore struct {
	path string
}

// NewFileStore creates a FileStore at the given path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Path returns the credential file path.
func (s *FileStore) Path() string {
	return s.path
}

// Load reads credentials from the JSON file.
// Returns (nil, nil) if the file does not exist.
func (s *FileStore) Load() (*Credentials, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	return &c, nil
}

// Save writes credentials as JSON with 0600 permissions.
func (s *FileStore) Save(c *Credentials) error {
	dir := filepath.Dir(s.path)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	return nil
}

// Delete removes the credential file.
// Returns nil if the file does not exist (idempotent).
func (s *FileStore) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete credentials: %w", err)
	}

	return nil
}

// EncryptedStore wraps a FileStore and encrypts content with AES-256-GCM.
// The key is derived from a password using Argon2id.
// File format: magic(5) || salt(16) || nonce(12) || ciphertext.
type EncryptedStore struct {
	inner    *FileStore
	password []byte
}

// NewEncryptedStore creates an EncryptedStore backed by inner.
func NewEncryptedStore(inner *FileStore, password []byte) *EncryptedStore {
	return &EncryptedStore{
		inner:    inner,
		password: password,
	}
}

// Load reads and decrypts credentials from the file.
// Returns (nil, nil) if the file does not exist.
func (s *EncryptedStore) Load() (*Credentials, error) {
	raw, err := os.ReadFile(s.inner.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read encrypted credentials: %w", err)
	}

	minLen := len(magicHeader) + saltLen + nonceLen + 1
	if len(raw) < minLen {
		return nil, fmt.Errorf("decrypt credentials: file too short")
	}

	offset := len(magicHeader)
	salt := raw[offset : offset+saltLen]
	nonce := raw[offset+saltLen : offset+saltLen+nonceLen]
	ciphertext := raw[offset+saltLen+nonceLen:]

	key := deriveKey(s.password, salt)

	plaintext, err := aesgcmDecrypt(key, nonce, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	var c Credentials
	if err := json.Unmarshal(plaintext, &c); err != nil {
		return nil, fmt.Errorf("parse decrypted credentials: %w", err)
	}

	return &c, nil
}

// Save encrypts and writes credentials to the file.
func (s *EncryptedStore) Save(c *Credentials) error {
	dir := filepath.Dir(s.inner.path)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	plaintext, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	salt, err := randBytes(saltLen)
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	nonce, err := randBytes(nonceLen)
	if err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	key := deriveKey(s.password, salt)

	ciphertext, err := aesgcmEncrypt(key, nonce, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	// magic || salt || nonce || ciphertext.
	out := make([]byte, 0, len(magicHeader)+saltLen+nonceLen+len(ciphertext))
	out = append(out, magicHeader...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	if err := os.WriteFile(s.inner.path, out, 0o600); err != nil {
		return fmt.Errorf("write encrypted credentials: %w", err)
	}

	return nil
}

// Delete removes the credential file.
func (s *EncryptedStore) Delete() error {
	return s.inner.Delete()
}

// IsEncrypted checks whether the file at path starts with the ERENC magic header.
// Returns false if the file does not exist or cannot be read.
func IsEncrypted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, len(magicHeader))

	n, err := io.ReadFull(f, header)
	if err != nil || n != len(magicHeader) {
		return false
	}

	return string(header) == string(magicHeader)
}

// TokenFromEnv returns the API key from EASY_RAILWAY_API_KEY, or empty if unset.
func TokenFromEnv() string {
	return os.Getenv("EASY_RAILWAY_API_KEY")
}

// ResolvePath returns the credentials file path using the priority order:
//  1. flagValue — from --credentials-path | -cdp flag (empty = not set).
//  2. EASY_RAILWAY_CREDENTIALS_PATH env var.
//  3. ~/.easy-railway/credentials.json (default).
func ResolvePath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if v := os.Getenv("EASY_RAILWAY_CREDENTIALS_PATH"); v != "" {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve credentials path: %w", err)
	}

	return filepath.Join(home, ".easy-railway", "credentials.json"), nil
}

// Crypto helpers.

func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func aesgcmEncrypt(key, nonce, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

func aesgcmDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("wrong password or corrupted file: %w", err)
	}

	return plaintext, nil
}

func randBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}

	return b, nil
}
