package logging

type SimpleLogger interface {
	Println(v ...interface{})
	Debugf(v ...interface{})
	Infof(v ...interface{})
	Warnf(v ...interface{})
	Errorf(v ...interface{})
	Panicf(v ...interface{})
	Fatalf(v ...interface{})
}
