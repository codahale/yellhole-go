package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

type config struct {
	Addr        string
	BaseURL     *url.URL
	DataDir     string
	Title       string
	Description string
	Author      string
	requestLog  bool
}

func parseConfig() (*config, error) {
	var config config

	var baseURL string
	cmd := flag.NewFlagSet("yellhole", flag.ContinueOnError)
	cmd.StringVar(&config.Addr, "addr", env("ADDR", "127.0.0.1:3000"), "the address on which to listen")
	cmd.StringVar(&baseURL, "base_url", env("BASE_URL", "http://localhost:3000"), "the base URL of the server")
	cmd.StringVar(&config.DataDir, "data_dir", env("DATA_DIR", "./data"), "the directory in which all persistent data is stored")
	cmd.StringVar(&config.Title, "title", env("TITLE", "Yellhole"), "the title of the yellhole instance")
	cmd.StringVar(&config.Description, "description", env("DESCRIPTION", "Obscurantist filth."), "the description of the yellhole instance")
	cmd.StringVar(&config.Author, "author", env("AUTHOR", "Luther Blissett"), "the author of the yellhole instance")
	cmd.BoolVar(&config.requestLog, "request_log", true, "enable logging requests to stdout")

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	config.BaseURL = u.JoinPath("./") // ensure we have a final slash

	if err := cmd.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	return &config, nil
}

func env(key, defaultValue string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return s
}
