package json

import "github.com/alfreddobradi/authd/config"

var (
	path string = "./auth.json"
)

func SetFromConfig(j config.JsonConfig) {
	if j.Path != "" {
		SetPath(j.Path)
	}
}

func Path() string {
	return path
}

func SetPath(newPath string) {
	path = newPath
}
