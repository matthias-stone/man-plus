package dictionary

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

type Client struct {
	appID, apiKey string
	baseURL       string
}

func NewClient(appID, APIKey, rootURL string) *Client {
	return &Client{appID, APIKey, rootURL}
}

type definitions struct {
	Results []struct {
		ID             string `json:"id"`
		Language       string `json:"language"`
		LexicalEntries []struct {
			Entries []struct {
				Etymologies         []string `json:"etymologies"`
				GrammaticalFeatures []struct {
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"grammaticalFeatures"`
				HomographNumber string `json:"homographNumber"`
				Notes           []struct {
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"notes"`
				Senses []struct {
					Definitions []string `json:"definitions"`
					Domains     []string `json:"domains,omitempty"`
					Examples    []struct {
						Text string `json:"text"`
					} `json:"examples,omitempty"`
					ID        string `json:"id"`
					Subsenses []struct {
						Definitions []string `json:"definitions"`
						Domains     []string `json:"domains,omitempty"`
						ID          string   `json:"id"`
						Examples    []struct {
							Text string `json:"text"`
						} `json:"examples,omitempty"`
					} `json:"subsenses,omitempty"`
					Regions []string `json:"regions,omitempty"`
				} `json:"senses"`
			} `json:"entries"`
			Language        string `json:"language"`
			LexicalCategory string `json:"lexicalCategory"`
			Pronunciations  []struct {
				AudioFile        string   `json:"audioFile"`
				Dialects         []string `json:"dialects"`
				PhoneticNotation string   `json:"phoneticNotation"`
				PhoneticSpelling string   `json:"phoneticSpelling"`
			} `json:"pronunciations"`
			Text string `json:"text"`
		} `json:"lexicalEntries"`
		Type string `json:"type"`
		Word string `json:"word"`
	} `json:"results"`
}

func (c *Client) LookupDictionaryWord(word string, w io.Writer) error {
	if word == "" {
		return errors.New("empty word provided, cannot perform dictionary lookup")
	}

	wordID, err := c.findWord(word)
	if err != nil {
		return errors.Wrap(err, "not found")
	}

	definitions, err := c.fetchDefinitions(wordID)
	if err != nil {
		return errors.Wrap(err, "could not load definitions")
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		fmt.Fprintln(pw, ".TH "+word+" "+definitions.Results[0].Language+` "Definitions (man-plus)" man-plus "Definitions (man-plus)"`)
		fmt.Fprintln(pw, ".SH DEFINITIONS")
		for _, le := range definitions.Results[0].LexicalEntries {
			for _, def := range le.Entries[0].Senses[0].Definitions {
				fmt.Fprintln(pw, "")
				fmt.Fprintln(pw, def)
			}
		}
	}()

	cmd := exec.Command("man", "/dev/stdin")

	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	return nil
}

// Returns the word_id from the API, or error if there were no results.
func (c *Client) findWord(word string) (string, error) {
	r := struct{ Results []struct{ Word, ID string } }{}
	params := map[string]string{
		"q":      word,
		"prefix": "false",
		"limit":  "1",
	}
	err := c.request("/search/en", params, &r)
	switch {
	case err != nil:
		return "", err
	case len(r.Results) != 1:
		return "", errors.New("no matches for word")
	}
	return r.Results[0].ID, nil
}

func (c *Client) fetchDefinitions(wordID string) (definitions, error) {
	var defs definitions
	return defs, c.request("/entries/en/"+wordID, nil, &defs)
}

// request submits a request to the Oxford Dictionaries API and parses it as JSON into the given object.
func (c *Client) request(path string, parameters map[string]string, v interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("app_id", c.appID)
	req.Header.Add("app_key", c.apiKey)

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
