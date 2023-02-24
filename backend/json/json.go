package json

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

type JsonAuthenticator struct {
	entries map[string]string
}

func Build() (*JsonAuthenticator, error) {
	fp, err := os.OpenFile(Path(), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", Path(), err)
	}

	entries := make(map[string]string)
	decoder := json.NewDecoder(fp)
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", Path(), err)
	}

	return &JsonAuthenticator{entries: entries}, nil
}

func (j *JsonAuthenticator) BasicAuth(r *http.Request) (string, error) {
	u, p, ok := r.BasicAuth()
	if !ok {
		return "", fmt.Errorf("no auth data")
	}

	hash, ok := j.entries[u]
	if !ok {
		return "", fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)); err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}

	return u, nil
}
