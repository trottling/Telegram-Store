package logger

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewFxLogger — служебные логи fx (PROVIDE/INVOKE/START/STOP) идут через тот
// же zap-логгер на уровне Debug: тихо в проде, видно при LOG_LEVEL=debug.
//
// Фатальный сбой старта сюда не полагается: fx.New возвращает такую ошибку
// через app.Err(), и каждый cmd/* печатает её сам в stderr — см. main.go.
func NewFxLogger(l *zap.SugaredLogger) fxevent.Logger {
	zapLogger := &fxevent.ZapLogger{Logger: l.Desugar()}
	zapLogger.UseLogLevel(zapcore.DebugLevel)
	return zapLogger
}
