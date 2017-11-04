package dictionary

import (
	"encoding/json"
	"io"
	"net/http"

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
		LexicalEntries []struct {
			Entries []struct {
				Senses []struct {
					Definitions []string
				}
			}
		}
	}
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

	fmt.Fprintln(w, definitions.Results[0].LexicalEntries[0].Entries[0].Senses[0].Definitions[0])

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
