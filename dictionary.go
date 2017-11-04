package main

import (
	"io"
	"os"

	"github.com/burntsushi/toml"
	"github.com/pkg/errors"
)

// ConfigPath is the location of man-plus' config file.
var ConfigPath = os.ExpandEnv("$HOME/.config/man-plus.toml")

// Config values
var Config = struct {
	AppID, APIKey string
	URL           string
}{
	"", "",
	"https://od-api.oxforddictionaries.com/api/v1",
}

func lookupDictionaryWord(word string) error {
	if word == "" {
		return errors.New("empty word provided, cannot perform dictionary lookup")
	}

	err := loadConfig()
	if err != nil {
		return errors.Wrap(err, "could not load apikey")
	}

	return nil
}

func loadConfig() error {
	_, err := toml.DecodeFile(ConfigPath, &Config)
	switch {
	case err != nil:
		return err
	case Config.AppID == "":
		return errors.New("config value AppID must be specified in the config file: " + ConfigPath)
	case Config.AppID == "":
		return errors.New("config value APIKey must be specified in the config file: " + ConfigPath)
	}
	return nil
}
