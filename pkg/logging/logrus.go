/**
 * log.go - logging wrapper
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package logging

import (
	"io/ioutil"
	"log/syslog"
	"os"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	lSyslog "github.com/Sirupsen/logrus/hooks/syslog"
)

const (
	outputSyslog = "syslog"
)

// Config log config
type Config struct {
	Level  string `toml:"level" json:"level"`
	Output string `toml:"output" json:"output"`
}

var log *logrus.Logger
var logoutput string
var lock = new(sync.Mutex)

// Configure sets up logging
func Configure(output string, l string) {
	lock.Lock()
	defer lock.Unlock()
	log = logrus.New()
	logoutput = output

	if output == "" || output == "stdout" {
		log.Out = os.Stdout
		log.Formatter = &logrus.TextFormatter{DisableColors: false, DisableTimestamp: false, QuoteEmptyFields: true}

	} else if output == "stderr" {
		log.Out = os.Stderr
		log.Formatter = &logrus.TextFormatter{DisableColors: false, DisableTimestamp: false, QuoteEmptyFields: true}

	} else if output == outputSyslog {
		log.Formatter = &logrus.TextFormatter{DisableColors: true, DisableTimestamp: true, QuoteEmptyFields: true}
		hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_LOCAL5, "")

		if err == nil {
			log.Hooks.Add(hook)
			log.Out = ioutil.Discard
		}

	} else {
		log.Formatter = &logrus.TextFormatter{DisableColors: true, DisableTimestamp: false, QuoteEmptyFields: true}
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			logrus.Fatal(err)
		}
		log.Out = f
	}

	if l == "" {
		return
	}

	if level, err := logrus.ParseLevel(l); err != nil {
		logrus.Fatal("Unknown loglevel ", l)
	} else {
		log.Level = level
	}
}

// For sets up logging defaults
func For(name string) *logrus.Entry {
	lock.Lock()
	defer lock.Unlock()
	tag := strings.Split(name, "/")
	return log.WithField("tag", name).WithField("func", tag[0])
}
