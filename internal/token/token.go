package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Generate() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func EnsureFile(path string) (value string, created bool, err error) {
	existing, err := LoadFile(path)
	if err == nil {
		return existing, false, nil
	}
	if !os.IsNotExist(err) {
		return "", false, err
	}

	value, err = Generate()
	if err != nil {
		return "", false, err
	}
	if err := writeProtected(path, value); err != nil {
		return "", false, err
	}
	return value, true, nil
}

func LoadFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(body))
	if value == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return value, nil
}

func RotateFile(path string) (string, error) {
	value, err := Generate()
	if err != nil {
		return "", err
	}
	if err := writeProtected(path, value); err != nil {
		return "", err
	}
	return value, nil
}

func writeProtected(path string, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".token-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(value + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
