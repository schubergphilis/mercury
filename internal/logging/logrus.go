package logging

import "github.com/sirupsen/logrus"

type Logrus struct {
	Logger *logrus.Logger
}

func (f *Logrus) Println(v ...interface{}) {
	f.Logger.Println(v...)
}

func (f *Logrus) Debugf(v ...interface{}) {
	f.Logger.Debugf(v[0].(string), v[1:]...)
}

func (f *Logrus) Infof(v ...interface{}) {
	f.Logger.Infof(v[0].(string), v[1:]...)
}

func (f *Logrus) Warnf(v ...interface{}) {
	f.Logger.Warnf(v[0].(string), v[1:]...)
}

func (f *Logrus) Errorf(v ...interface{}) {
	f.Logger.Errorf(v[0].(string), v[1:]...)
}

func (f *Logrus) Fatalf(v ...interface{}) {
	f.Logger.Fatalf(v[0].(string), v[1:]...)
}

func (f *Logrus) Panicf(v ...interface{}) {
	f.Logger.Panicf(v[0].(string), v[1:]...)
}

func NewLogrus() (SimpleLogger, error) {
	return &Logrus{
		Logger: logrus.New(),
	}, nil
}
