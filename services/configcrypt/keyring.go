// Package configcrypt provides versioned envelope encryption for configuration
// center payloads and keeps key-encryption keys outside the application data.
package configcrypt

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const (
	ProviderFile   = "file"
	keyBytes       = 32
	keyringVersion = 1
	maxKeyringSize = 64 * 1024
	osWindows      = "windows"
)

var (
	ErrDisabled         = errors.New("configuration encryption is disabled")
	ErrInvalidSettings  = errors.New("invalid configuration encryption settings")
	ErrInvalidKeyring   = errors.New("invalid configuration encryption keyring")
	ErrInsecureKeyring  = errors.New("configuration encryption keyring permissions are insecure")
	ErrKeyNotFound      = errors.New("configuration encryption key not found")
	ErrEncrypt          = errors.New("configuration encryption failed")
	ErrDecrypt          = errors.New("configuration decryption failed")
	ErrCipherSetup      = errors.New("configuration cipher setup failed")
	ErrKeyAlreadyExists = errors.New("configuration encryption key already exists")
	ErrKeyringWrite     = errors.New("configuration encryption keyring write failed")
	keyIDPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

type Settings struct {
	Provider    string
	ActiveKeyID string
	KeyringPath string
	Enabled     bool
}

type keyringDocument struct {
	Keys    map[string]string `json:"keys"`
	Version int               `json:"version"`
}

func loadFileKeys(path string) (map[string][]byte, error) {
	document, err := readKeyring(path)
	if err != nil {
		return nil, err
	}
	keys := make(map[string][]byte, len(document.Keys))
	for id, encoded := range document.Keys {
		if !keyIDPattern.MatchString(id) {
			return nil, ErrInvalidKeyring
		}
		key, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(key) != keyBytes {
			return nil, ErrInvalidKeyring
		}
		keys[id] = key
	}
	return keys, nil
}

func readKeyring(path string) (*keyringDocument, error) {
	if path == "" || !filepath.IsAbs(path) {
		return nil, ErrInvalidSettings
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.Join(ErrInvalidKeyring, err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxKeyringSize {
		return nil, ErrInvalidKeyring
	}
	if runtime.GOOS != osWindows && info.Mode().Perm()&0o077 != 0 {
		return nil, ErrInsecureKeyring
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Join(ErrInvalidKeyring, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document keyringDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, errors.Join(ErrInvalidKeyring, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, ErrInvalidKeyring
		}
		return nil, errors.Join(ErrInvalidKeyring, err)
	}
	if document.Version != keyringVersion || len(document.Keys) == 0 {
		return nil, ErrInvalidKeyring
	}
	return &document, nil
}

// GenerateFileKey creates a new AES-256 key and stores it in a separate
// keyring file. Existing keys are retained so older ciphertext stays readable.
func GenerateFileKey(path, id string) error {
	if path == "" || !filepath.IsAbs(path) || !keyIDPattern.MatchString(id) {
		return ErrInvalidSettings
	}
	document := &keyringDocument{Version: keyringVersion, Keys: make(map[string]string)}
	if _, err := os.Stat(path); err == nil {
		existing, readErr := readKeyring(path)
		if readErr != nil {
			return readErr
		}
		document = existing
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.Join(ErrKeyringWrite, err)
	}
	if _, exists := document.Keys[id]; exists {
		return ErrKeyAlreadyExists
	}
	key := make([]byte, keyBytes)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return errors.Join(ErrKeyringWrite, err)
	}
	document.Keys[id] = base64.StdEncoding.EncodeToString(key)
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return errors.Join(ErrKeyringWrite, err)
	}
	raw = append(raw, '\n')
	if err := writeKeyringAtomically(path, raw); err != nil {
		return err
	}
	return nil
}

func writeKeyringAtomically(path string, raw []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".config-center-keyring-*")
	if err != nil {
		return errors.Join(ErrKeyringWrite, err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return errors.Join(ErrKeyringWrite, err)
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return errors.Join(ErrKeyringWrite, err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return errors.Join(ErrKeyringWrite, err)
	}
	if err := temporary.Close(); err != nil {
		return errors.Join(ErrKeyringWrite, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return errors.Join(ErrKeyringWrite, err)
	}
	return nil
}
