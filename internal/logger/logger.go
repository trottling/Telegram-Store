package logger

import (
	"github.com/trottling/Telegram-Store/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New — SugaredLogger, не строгий *zap.Logger: у него есть и Infow(msg, "k", v)
// под структурные вызовы, и Infof(fmt, args...) под printf-стиль — обе формы
// используются по всему репо, а имена методов совпадают буква в букву с
// прежними logrus-вызовами там, где формат уже был printf-style.
func New(c *config.LoggerConfig) *zap.SugaredLogger {
	var unknownLevel bool

	var level zapcore.Level
	switch c.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	case "panic":
		level = zapcore.PanicLevel
	default:
		level = zapcore.InfoLevel
		unknownLevel = c.Level != ""
	}

	// Caller/stacktrace только при debug
	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		DisableCaller:     level != zapcore.DebugLevel,
		DisableStacktrace: level != zapcore.DebugLevel,
		Encoding:          "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			MessageKey:     "msg",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Ошибка тут — по сути только кривой OutputPaths, которого нет: конфиг
	// собирается из констант выше, а не из чего-то, что реально может подвести.
	l, err := cfg.Build()
	if err != nil {
		panic("logger: failed to build zap logger: " + err.Error())
	}
	sugar := l.Sugar()

	// zap.S()/zap.L() (глобальные, без DI) иначе no-op по умолчанию — в отличие
	// от вечно рабочего logrus.StandardLogger(), на который раньше опирался
	// bot/texts.T(): без этого редкий, но реальный лог о битом переводе просто
	// исчезал бы молча. Один настроенный логгер на процесс — тот же самый,
	// что уходит в fx и во все репозитории/сервисы через DI.
	zap.ReplaceGlobals(l)

	// Предупреждаем уже настроенным логгером: молчаливый откат на info означал,
	// что опечатка вроде LOG_LEVEL=warning даёт уровень болтливее ожидаемого, и
	// заметить это можно только по объёму логов.
	if unknownLevel {
		sugar.Warnw("logger: unrecognized log level, falling back to info", "log_level", c.Level)
	}

	return sugar
}
