package yellhole

import (
	"net/http"

	"github.com/codahale/yellhole-go/internal/static"
)

type App struct {
	config *Config
}

func NewApp(config *Config) App {
	return App{config}
}

func (app *App) Handler() http.Handler {
	mux := http.NewServeMux()
	static.Register(mux)
	return mux
}
