package logging

import "testing"

func TestStandardLogger(t *testing.T) {
	Configure("stdout", "debug")
	l := StandardLog("test/log")
	l.Print("log entry\n")
}
