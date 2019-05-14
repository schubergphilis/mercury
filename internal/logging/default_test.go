package logging

import (
	"log"
	"testing"
)

func TestDefault(t *testing.T) {
	logger, _ := NewDefault()
	log.Printf("logger: %v", logger.Logger)
	logger.Infof("info log")

}
