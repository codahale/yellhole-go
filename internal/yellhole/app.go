package yellhole

import (
	"net/http"

	"github.com/codahale/yellhole-go/internal/assets"
)

type App struct {
}

func NewApp(config *Config) App {
	return App{}
}

func (app *App) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	assets.Register(mux)
	return mux
}
