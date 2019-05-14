package logging

import (
	"testing"
)

func TestWrapper(t *testing.T) {
	logger, _ := NewZap()
	logger.Infof("info log")

	var prefix []interface{}
	prefix = append(prefix, "test")

	wrapper := Wrapper{
		Log:    logger,
		Level:  ErrorLevel,
		Prefix: prefix,
	}
	wrapper.Infof("info log")
	wrapper.Warnf("warn log")
}
