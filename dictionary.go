package main

import "errors"

func lookupDictionaryWord(word string) error {
	if word == "" {
		return errors.New("empty word provided, cannot perform dictionary lookup")
	}
	return nil
}
