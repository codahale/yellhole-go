package yellhole

import (
	"net/http"

	"github.com/codahale/yellhole-go/internal/yellhole/config"
	"github.com/codahale/yellhole-go/internal/yellhole/static"
)

type Config = config.Config

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

func ParseConfig() (*Config, error) {
	return config.ParseConfig()
}
