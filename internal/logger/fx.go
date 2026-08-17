package logger

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/fx/fxevent"
)

// NewFxLogger — служебные логи fx (PROVIDE/INVOKE/START/STOP) идут через тот
// же logrus-логгер на уровне Debug: тихо в проде, видно при LOG_LEVEL=debug.
func NewFxLogger(l *logrus.Logger) fxevent.Logger {
	return &fxevent.ConsoleLogger{W: l.WriterLevel(logrus.DebugLevel)}
}
