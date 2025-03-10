package yellhole

import (
	"flag"
	"os"
)

type Config struct {
	Addr        string
	BaseURL     string
	DataDir     string
	Title       string
	Description string
	Author      string
}

func ParseConfig() (*Config, error) {
	var config Config

	cmd := flag.NewFlagSet("yellhole", flag.ContinueOnError)
	cmd.StringVar(&config.Addr, "addr", "127.0.0.1:3000", "the address on which to listen")
	cmd.StringVar(&config.BaseURL, "base_url", "http://127.0.0.1:3000/", "the base URL of the server")
	cmd.StringVar(&config.DataDir, "data_dir", "./data", "the directory in which all persistent data is stored")
	cmd.StringVar(&config.Title, "title", "Yellhole", "the title of the yellhole instance")
	cmd.StringVar(&config.Description, "description", "Obscurantist filth.", "the description of the yellhole instance")
	cmd.StringVar(&config.Author, "author", "Luther Blissett", "the author of the yellhole instance")

	if addr, ok := os.LookupEnv("ADDR"); ok {
		config.Addr = addr
	}

	if baseURL, ok := os.LookupEnv("BASE_URL"); ok {
		config.BaseURL = baseURL
	}

	if dataDir, ok := os.LookupEnv("DATA_DIR"); ok {
		config.DataDir = dataDir
	}

	if title, ok := os.LookupEnv("TITLE"); ok {
		config.Title = title
	}

	if description, ok := os.LookupEnv("DESCRIPTION"); ok {
		config.Description = description
	}

	if author, ok := os.LookupEnv("AUTHOR"); ok {
		config.Author = author
	}

	if err := cmd.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	return &config, nil
}
