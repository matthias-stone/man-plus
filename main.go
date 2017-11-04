// Package man-plus provides a command line tool that wraps the man command and
// looks up dictionary definitions of words that are not found.
//
// A simple way to deploy:
//    alias man=man-plus
//
// man-plus requires an API key for the Oxford Dictionaries. A user is required
// provide their own app ID and key, available at https://developer.oxforddictionaries.com/.
// Once acquired the values should be stored in the file ~/.config/man-plus.toml
// with the following format:
//     AppID = "MyAppID"
//     APIKey = "MyAPIKey"
//
package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/burntsushi/toml"
	"github.com/matthias-stone/man-plus/dictionary"
)

// ManPageNotFoundExitCode is the exit code man returns if the page is not found.
// Determined by experimentation.
const ManPageNotFoundExitCode = 4096

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

func init() {
	log.SetFlags(0)
}

func main() {
	cmd := exec.Command("man", os.Args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	errbuf := bufio.NewWriter(os.Stderr)
	cmd.Stderr = errbuf
	defer errbuf.Flush()

	err := cmd.Run()
	if err == nil {
		os.Exit(0)
	}

	exit, ok := err.(*exec.ExitError)
	if !ok {
		log.Fatal("Unknown error:", err.Error())
	}

	status, ok := exit.Sys().(syscall.WaitStatus)
	if !ok {
		log.Fatal("Wait status not available")
	}

	// Only do dictionary lookups if there were no arguments and the command was not found.
	if len(os.Args) != 2 || status != ManPageNotFoundExitCode {
		os.Exit(int(status))
	}

	errbuf.Reset(os.Stderr)
	if err := loadConfig(); err != nil {
		log.Fatalf("Unable to load API credentials for Oxford Dictionaries: %s", err.Error())
	}

	client := dictionary.NewClient(Config.AppID, Config.APIKey, Config.URL)

	// Perform a dictionary lookup on the one word that we found.
	word := os.Args[1]
	err = client.LookupDictionaryWord(word, os.Stdout)
	if err != nil {
		log.Fatalf("Could not find dictionary word for '%s': %s", word, err.Error())
	}
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
