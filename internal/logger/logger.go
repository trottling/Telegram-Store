package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/config"
)

func New(c *config.LoggerConfig) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)

	var level logrus.Level
	switch c.Level {
	case "debug":
		level = logrus.DebugLevel
	case "info":
		level = logrus.InfoLevel
	case "warn":
		level = logrus.WarnLevel
	case "error":
		level = logrus.ErrorLevel
	case "fatal":
		level = logrus.FatalLevel
	case "panic":
		level = logrus.PanicLevel
	default:
		level = logrus.InfoLevel

	}

	l.SetLevel(level)
	l.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		DisableLevelTruncation: true,
		FullTimestamp:          true,
	})

	return l
}
