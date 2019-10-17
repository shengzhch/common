package zaplog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type Logger struct {
	l *zap.SugaredLogger
}

var defaultLoger *Logger

func FormatTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func init() {
	atom := zap.NewAtomicLevel()
	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev >= zap.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev < zap.ErrorLevel && lev > zap.DebugLevel
	})

	prodEncoder := zap.NewDevelopmentEncoderConfig()
	prodEncoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
	prodEncoder.EncodeTime = FormatTimeEncoder

	console := zapcore.Lock(os.Stdout)
	lowWriteSyncer, lowClose, err := zap.Open("./zaplog/err.log")
	if err != nil {
		lowClose()
		panic(err)
	}
	highWriteSyncer, highClose, err := zap.Open("./zaplog/info.log")
	if err != nil {
		highClose()
		panic(err)
	}
	hcore := zapcore.NewCore(zapcore.NewJSONEncoder(prodEncoder), highWriteSyncer, highPriority)
	lcore := zapcore.NewCore(zapcore.NewJSONEncoder(prodEncoder), lowWriteSyncer, lowPriority)
	ccore := zapcore.NewCore(zapcore.NewConsoleEncoder(prodEncoder), console, atom)

	core := zapcore.NewTee(ccore, hcore, lcore)

	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(highPriority))
	defaultLoger = &Logger{l.Sugar()}
}

func Debug(args ...interface{}) {
	defaultLoger.l.Debug(args...)
}

func Info(args ...interface{}) {
	defaultLoger.l.Info(args...)
}

func Warn(args ...interface{}) {
	defaultLoger.l.Warn(args...)
}

func Error(args ...interface{}) {
	defaultLoger.l.Error(args...)
}

func Fatal(args ...interface{}) {
	defaultLoger.l.Fatal(args...)
}

func Debugf(template string, args ...interface{}) {
	defaultLoger.l.Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	defaultLoger.l.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	defaultLoger.l.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	defaultLoger.l.Errorf(template, args...)
}
