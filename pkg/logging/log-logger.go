package logging

import (
	stdlog "log"
)

// StandardLogWriterInterface is an interface to wrap log.Logger
type StandardLogWriterInterface interface {
	Write(p []byte) (n int, err error)
}

// StandardLogWrite implements StandardLogWriterInterface
type StandardLogWrite struct {
	//buf  bytes.Buffer
	name string
}

// Write is the write handler for StandardLog
func (s StandardLogWrite) Write(p []byte) (n int, err error) {
	str := string(p)
	For(s.name).Warn(str)
	return len(p), nil
}

// StandardLog returns a log.Logger wrapper
func StandardLog(name string) *stdlog.Logger {
	w := StandardLogWrite{name: name}
	std := stdlog.New(w, "", stdlog.Lshortfile)
	return std
}
