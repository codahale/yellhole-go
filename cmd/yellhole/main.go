package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/codahale/yellhole-go/internal/yellhole"
)

func main() {
	config, err := yellhole.ParseConfig()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		} else {
			os.Exit(2)
		}
	}

	app := yellhole.NewApp(config)
	mux := app.Mux()
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalln("error serving HTTP", err)
	}
}
