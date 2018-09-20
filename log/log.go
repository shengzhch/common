package log

import "go.uber.org/zap"

type Logger struct {
	*zap.SugaredLogger
}

var (
	logger *Logger
)

func init() {
	dp, _ := zap.NewDevelopment()
	dl := dp.Sugar()
	logger = &Logger{dl}
}

func GetALogger() *Logger {
	return logger

}
