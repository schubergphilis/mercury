package logging

import (
	stdlog "log"
)

type StandardLogWriterInterface interface {
	Write(p []byte) (n int, err error)
}

type StandardLogWrite struct {
	//buf  bytes.Buffer
	name string
}

func (s StandardLogWrite) Write(p []byte) (n int, err error) {
	str := string(p)
	For(s.name).Warn(str)
	return len(p), nil
}

func StandardLog(name string) *stdlog.Logger {
	w := StandardLogWrite{name: name}
	std := stdlog.New(w, "", stdlog.Lshortfile)
	return std
}
