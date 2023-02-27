package json

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
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

func (j *JsonAuthenticator) BasicAuth(ctx context.Context, r *http.Request) (string, error) {
	_, span := otel.Tracer("auth").Start(ctx, "json")
	defer span.End()
	u, p, ok := r.BasicAuth()
	if !ok {
		err := fmt.Errorf("no auth data")
		span.RecordError(err)
		return "", err
	}

	hash, ok := j.entries[u]
	if !ok {
		err := fmt.Errorf("user not found")
		span.RecordError(err)
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)); err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("bcrypt: %w", err)
	}

	return u, nil
}
