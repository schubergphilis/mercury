package logging

import (
	"go.uber.org/zap"
)

type ZapSugar struct {
	Logger *zap.SugaredLogger
	sync   func() error
}

func (f *ZapSugar) Println(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Infow(v[0].(string), v[1:]...)
	case error:
		f.Logger.Infow(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Infow("Info", v...)
	}
}

func (f *ZapSugar) Debugf(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Debugw(v[0].(string), v[1:]...)
	case error:
		f.Logger.Debugw(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Debugw("Debug", v...)
	}
}

func (f *ZapSugar) Infof(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Infow(v[0].(string), v[1:]...)
	case error:
		f.Logger.Infow(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Infow("Info", v...)
	}
}

func (f *ZapSugar) Warnf(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Warnw(v[0].(string), v[1:]...)
	case error:
		f.Logger.Warnw(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Warnw("Warn", v...)
	}
}

func (f *ZapSugar) Errorf(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Errorw(v[0].(string), v[1:]...)
	case error:
		f.Logger.Errorw(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Errorw("Error", v...)
	}
}

func (f *ZapSugar) Fatalf(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Fatalw(v[0].(string), v[1:]...)
	case error:
		f.Logger.Fatalw(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Fatalw("Fatal", v...)
	}
}

func (f *ZapSugar) Panicf(v ...interface{}) {
	switch v[0].(type) {
	case string:
		f.Logger.Panicw(v[0].(string), v[1:]...)
	case error:
		f.Logger.Panicw(v[0].(error).Error(), v[1:]...)
	default:
		f.Logger.Panicw("Panic", v...)
	}
}

func NewZap(dst ...string) (*ZapSugar, error) {
	config := zap.NewProductionConfig()
	if len(dst) == 0 {
		config.OutputPaths = []string{"stdout"}
	} else {
		config.OutputPaths = dst
	}
	config.DisableCaller = true
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "timestamp"
	//config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "message"
	config.Level.SetLevel(zap.DebugLevel)
	//cfg.Level = zap.DebugLevel
	logger, err := config.Build()
	return &ZapSugar{
		Logger: logger.Sugar(),
		sync:   logger.Sync,
	}, err
}
