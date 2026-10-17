package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Store handles encrypted persistence of cookies and session tokens.
type Store struct {
	mu      sync.RWMutex
	dir     string
	key     []byte
	gcm     cipher.AEAD
}

// NewStore initializes an encrypted session store in the specified directory.
// If secretKey is empty, a deterministic key derived from host identity is used.
func NewStore(dir string, secretKey string) (*Store, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".aigobrowser", "sessions")
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session store dir: %w", err)
	}

	if secretKey == "" {
		secretKey = os.Getenv("AIGOBROWSER_SESSION_KEY")
	}
	if secretKey == "" {
		// Fallback deterministic local key
		hostname, _ := os.Hostname()
		secretKey = "aigobrowser-default-seed-" + hostname
	}

	hash := sha256.Sum256([]byte(secretKey))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Store{
		dir: dir,
		key: hash[:],
		gcm: gcm,
	}, nil
}

// SaveSession encrypts and saves an arbitrary JSON-serializable session value.
func (s *Store) SaveSession(name string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := s.gcm.Seal(nonce, nonce, data, nil)
	filePath := filepath.Join(s.dir, name+".enc")
	return os.WriteFile(filePath, ciphertext, 0600)
}

// LoadSession decrypts and deserializes a previously saved session.
func (s *Store) LoadSession(name string, dest any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.dir, name+".enc")
	ciphertext, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	nonceSize := s.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := s.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	return json.Unmarshal(plaintext, dest)
}

// SaveCookies saves a list of http.Cookies for a given domain/service.
func (s *Store) SaveCookies(domain string, cookies []*http.Cookie) error {
	return s.SaveSession("cookies_"+domain, cookies)
}

// LoadCookies retrieves decrypted cookies for a given domain/service.
func (s *Store) LoadCookies(domain string) ([]*http.Cookie, error) {
	var cookies []*http.Cookie
	err := s.LoadSession("cookies_"+domain, &cookies)
	if err != nil {
		return nil, err
	}
	return cookies, nil
}
