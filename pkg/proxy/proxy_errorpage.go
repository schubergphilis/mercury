package proxy

import (
	"io/ioutil"
)

// ErrorPage contains the page to show on errors
type ErrorPage struct {
	File             string `json:"file" toml:"file"`                           // alternative error page to show
	StatusCode       int    `json:"statuscode" toml:"statuscode"`               // error code to give
	StatusMessage    string `json:"statusmessage" toml:"statusmessage"`         // error message to apply
	TriggerThreshold int    `json:"trigger_threshold" toml:"trigger_threshold"` // Theshold at which to trigger the error page (generally 500 and up)
	content          []byte
}

// load Loads the file to contents
func (e *ErrorPage) load() (err error) {
	if e.File == "" {
		e.content = []byte{}
		return nil
	}
	e.content, err = ioutil.ReadFile(e.File)
	return err
}

// present returns true if there is a sorry page
func (e *ErrorPage) present() bool {
	return len(e.content) > 0
}

// threshold returns true if status reached the threshold
func (e *ErrorPage) threshold(status int) bool {
	return status >= e.TriggerThreshold
}
