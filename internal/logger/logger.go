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
	// ForceColors намеренно не выставляем: logrus сам включит цвета, только
	// если stdout — терминал. В контейнере это pipe, и escape-последовательности
	// уходили бы в json-file прямо в поле log, отравляя и `docker logs`, и
	// любой агрегатор.
	l.SetFormatter(&logrus.TextFormatter{
		DisableLevelTruncation: true,
		FullTimestamp:          true,
	})

	return l
}
