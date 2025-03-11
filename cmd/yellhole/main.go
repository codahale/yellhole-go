package main

import (
	"errors"
	"flag"
	"fmt"
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
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	app := yellhole.NewApp(config)
	if err := http.ListenAndServe(":8080", app.Handler()); err != nil {
		log.Fatalln("error serving HTTP", err)
	}
}
