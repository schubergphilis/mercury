package logging

import (
	"log"
	"os"
)

type Default struct {
	Logger *log.Logger
}

func (f *Default) Println(v ...interface{}) {
	log.Printf("logger: %+v %T", f.Logger, f.Logger)
	f.Logger.Println(v...)
}

func (f *Default) Debugf(v ...interface{}) {
	f.Logger.Println("DEBUG:", v)
}

func (f *Default) Infof(v ...interface{}) {
	f.Logger.Println("INFO:", v)
}

func (f *Default) Warnf(v ...interface{}) {
	f.Logger.Println("WARN:", v)
}

func (f *Default) Errorf(v ...interface{}) {
	f.Logger.Println("ERROR:", v)
}

func (f *Default) Fatalf(v ...interface{}) {
	f.Logger.Println("FATAL:", v)
}

func (f *Default) Panicf(v ...interface{}) {
	f.Logger.Println("PANIC:", v)
}

func NewDefault() (*Default, error) {
	l := &log.Logger{}
	l.SetOutput(os.Stderr)
	return &Default{
		//Logger: log.New(os.Stderr),
		Logger: l,
	}, nil

}
