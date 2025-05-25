package main

import (
	"flag"

	"github.com/Xuanwo/go-locale"
)

// loadConfig loads the app configuration from the command line arguments and environment variables.
func loadConfig(args []string, lookupEnv func(string) (string, bool)) (addr, baseURL, dataDir, author, title, description, lang string, err error) {
	env := func(key, defaultValue string) string {
		s, ok := lookupEnv(key)
		if !ok {
			return defaultValue
		}
		return s
	}

	detectedLang, err := locale.Detect()
	if err != nil {
		return
	}

	cmd := flag.NewFlagSet("yellhole", flag.ContinueOnError)
	cmd.StringVar(&addr, "addr", env("ADDR", "127.0.0.1:3000"), "the address on which to listen")
	cmd.StringVar(&baseURL, "base_url", env("BASE_URL", "http://localhost:3000/"), "the base URL of the server")
	cmd.StringVar(&dataDir, "data_dir", env("DATA_DIR", "./data"), "the directory in which all persistent data is stored")
	cmd.StringVar(&author, "author", env("AUTHOR", "Luther Blissett"), "the author of the yellhole instance")
	cmd.StringVar(&title, "title", env("TITLE", "Yellhole"), "the title of the yellhole instance")
	cmd.StringVar(&description, "description", env("DESCRIPTION", "Obscurantist filth."), "the description of the yellhole instance")
	cmd.StringVar(&lang, "lang", detectedLang.String(), "the language of the notes")

	err = cmd.Parse(args)
	return
}
