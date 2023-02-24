package backend

import "net/http"

type BasicAuthenticator interface {
	BasicAuth(r *http.Request) (string, error)
}
