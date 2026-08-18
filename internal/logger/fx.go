package logger

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/fx/fxevent"
)

// NewFxLogger — служебные логи fx (PROVIDE/INVOKE/START/STOP) идут через тот
// же logrus-логгер на уровне Debug: тихо в проде, видно при LOG_LEVEL=debug.
//
// Но события, несущие ошибку, поднимаются до Error. Иначе сбой старта уходил
// в Debug и при штатном LOG_LEVEL=info процесс просто завершался с кодом 1 без
// единой строки в логах — контейнер уходил в перезапуски, а причину видно не
// было. Заметно это не на всяком сбое: например, ошибку конфига fx печатает и
// так, потому что logrus сам зависит от *config.Config и при кривом конфиге
// просто не создаётся, так что fx откатывается на свой резервный логгер.
func NewFxLogger(l *logrus.Logger) fxevent.Logger {
	return &fxLogger{
		log:     l,
		console: &fxevent.ConsoleLogger{W: l.WriterLevel(logrus.DebugLevel)},
	}
}

type fxLogger struct {
	log     *logrus.Logger
	console fxevent.Logger
}

func (f *fxLogger) LogEvent(event fxevent.Event) {
	f.console.LogEvent(event)

	if err := eventErr(event); err != nil {
		f.log.WithError(err).Error("fx: lifecycle event failed")
	}
}

// eventErr — ошибка из события, если она там есть. Разбираем только те типы,
// что действительно её несут; всё прочее уже ушло в console на Debug.
func eventErr(event fxevent.Event) error {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		return e.Err
	case *fxevent.OnStopExecuted:
		return e.Err
	case *fxevent.Supplied:
		return e.Err
	case *fxevent.Provided:
		return e.Err
	case *fxevent.Replaced:
		return e.Err
	case *fxevent.Decorated:
		return e.Err
	case *fxevent.Invoked:
		return e.Err
	case *fxevent.Started:
		return e.Err
	case *fxevent.Stopped:
		return e.Err
	case *fxevent.RolledBack:
		return e.Err
	default:
		return nil
	}
}
