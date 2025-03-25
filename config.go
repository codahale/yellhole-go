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
	RequestLog  bool
}

func parseConfig() (*config, error) {
	var config config

	var baseURL string
	cmd := flag.NewFlagSet("yellhole", flag.ContinueOnError)
	cmd.StringVar(&config.Addr, "addr", "127.0.0.1:3000", "the address on which to listen")
	cmd.StringVar(&baseURL, "base_url", "http://localhost:3000", "the base URL of the server")
	cmd.StringVar(&config.DataDir, "data_dir", "./data", "the directory in which all persistent data is stored")
	cmd.StringVar(&config.Title, "title", "Yellhole", "the title of the yellhole instance")
	cmd.StringVar(&config.Description, "description", "Obscurantist filth.", "the description of the yellhole instance")
	cmd.StringVar(&config.Author, "author", "Luther Blissett", "the author of the yellhole instance")
	cmd.BoolVar(&config.RequestLog, "request_log", true, "enable logging requests to stdout")

	if s, ok := os.LookupEnv("ADDR"); ok {
		config.Addr = s
	}

	if s, ok := os.LookupEnv("BASE_URL"); ok {
		baseURL = s
	}

	if s, ok := os.LookupEnv("DATA_DIR"); ok {
		config.DataDir = s
	}

	if s, ok := os.LookupEnv("TITLE"); ok {
		config.Title = s
	}

	if s, ok := os.LookupEnv("DESCRIPTION"); ok {
		config.Description = s
	}

	if s, ok := os.LookupEnv("AUTHOR"); ok {
		config.Author = s
	}

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
