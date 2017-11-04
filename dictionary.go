package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type definitions struct {
	Results []struct {
		LexicalEntries []struct {
			Entries []struct {
				Senses []struct {
					Definitions []string
				}
			}
		}
	}
}

func lookupDictionaryWord(word string, w io.Writer) error {
	if word == "" {
		return errors.New("empty word provided, cannot perform dictionary lookup")
	}

	err := loadConfig()
	if err != nil {
		return errors.Wrap(err, "could not load apikey")
	}

	wordID, err := findWord(word)
	if err != nil {
		return errors.Wrap(err, "not found")
	}

	definitions, err := fetchDefinitions(wordID)
	if err != nil {
		return errors.Wrap(err, "could not load definitions")
	}

	fmt.Fprintln(w, definitions.Results[0].LexicalEntries[0].Entries[0].Senses[0].Definitions[0])

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

// Returns the word_id from the API, or error if there were no results.
func findWord(word string) (string, error) {
	r := struct{ Results []struct{ Word, ID string } }{}
	params := map[string]string{
		"q":      word,
		"prefix": "false",
		"limit":  "1",
	}
	err := request("/search/en", params, &r)
	switch {
	case err != nil:
		return "", err
	case len(r.Results) != 1:
		return "", errors.New("no matches for word")
	}
	return r.Results[0].ID, nil
}

func fetchDefinitions(wordID string) (definitions, error) {
	var defs definitions
	return defs, request("/entries/en/"+wordID, nil, &defs)
}

// request submits a request to the Oxfort Dictionaries API and parses it as JSON into the given object.
func request(path string, parameters map[string]string, v interface{}) error {
	req, err := http.NewRequest("GET", Config.URL+path, nil)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("app_id", Config.AppID)
	req.Header.Add("app_key", Config.APIKey)

	values := req.URL.Query()
	for key, value := range parameters {
		values.Add(key, value)
	}
	req.URL.RawQuery = values.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "connecting to "+req.URL.String())
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected " + resp.Status + " from " + req.URL.String())
	}

	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}
