package backend

import (
	"context"
	"net/http"
)

type BasicAuthenticator interface {
	BasicAuth(ctx context.Context, r *http.Request) (string, error)
}
