// Package dbstats — Prometheus-метрики пула соединений к Postgres
// (database/sql.DB.Stats()), общий для всех трёх Go-бинарников: у каждого
// свой *gorm.DB/пул (см. internal/repository/postgres.NewClient), поэтому
// один и тот же Collector подключается в cmd/bot, cmd/admin_backend и
// cmd/payments_backend по отдельности — разные процессы, разные registry,
// разные /metrics-эндпоинты, различаются в Prometheus по лейблу job (см.
// monitoring/prometheus/prometheus.yml).
package dbstats

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Collector — та же идиома, что и adminmetrics.AnalyticsCollector: значения
// считываются заново на каждый Prometheus-скрейп (sql.DB.Stats() — обычное
// чтение состояния пула, не запрос к БД), отдельной горутины на опрос не
// нужно.
type Collector struct {
	db  *gorm.DB
	log *zap.SugaredLogger

	maxOpenConnections *prometheus.Desc
	openConnections    *prometheus.Desc
	inUse              *prometheus.Desc
	idle               *prometheus.Desc
	waitCount          *prometheus.Desc
	waitDuration       *prometheus.Desc
	maxIdleClosed      *prometheus.Desc
	maxIdleTimeClosed  *prometheus.Desc
	maxLifetimeClosed  *prometheus.Desc
}

// New регистрирует себя в дефолтном registry прямо в конструкторе — тот
// вызывается через fx ровно один раз на процесс, повторной регистрации не
// будет (см. RunMetricsServer в каждом cmd/*, форсирующий сборку через
// неиспользуемый параметр).
func New(db *gorm.DB, log *zap.SugaredLogger) *Collector {
	c := &Collector{
		db:  db,
		log: log,

		maxOpenConnections: prometheus.NewDesc("db_max_open_connections", "Maximum number of open connections allowed to the database.", nil, nil),
		openConnections:    prometheus.NewDesc("db_open_connections", "The number of established connections, both in use and idle.", nil, nil),
		inUse:              prometheus.NewDesc("db_connections_in_use", "The number of connections currently in use.", nil, nil),
		idle:               prometheus.NewDesc("db_connections_idle", "The number of idle connections.", nil, nil),
		waitCount:          prometheus.NewDesc("db_wait_count_total", "The total number of connections waited for because none were free.", nil, nil),
		waitDuration:       prometheus.NewDesc("db_wait_duration_seconds_total", "The total time blocked waiting for a new connection.", nil, nil),
		maxIdleClosed:      prometheus.NewDesc("db_max_idle_closed_total", "The total number of connections closed due to SetMaxIdleConns.", nil, nil),
		maxIdleTimeClosed:  prometheus.NewDesc("db_max_idle_time_closed_total", "The total number of connections closed due to SetConnMaxIdleTime.", nil, nil),
		maxLifetimeClosed:  prometheus.NewDesc("db_max_lifetime_closed_total", "The total number of connections closed due to SetConnMaxLifetime.", nil, nil),
	}
	prometheus.MustRegister(c)
	return c
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenConnections
	ch <- c.openConnections
	ch <- c.inUse
	ch <- c.idle
	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxIdleTimeClosed
	ch <- c.maxLifetimeClosed
}

// Collect дергает db.DB() на каждый скрейп, а не кэширует *sql.DB в поле:
// он один и тот же на всё время жизни процесса, но так проще не думать о
// протухании после гипотетического Close.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	sqlDB, err := c.db.DB()
	if err != nil {
		c.log.Errorw("dbstats: failed to get underlying sql.DB", "error", err)
		return
	}
	stats := sqlDB.Stats()

	ch <- prometheus.MustNewConstMetric(c.maxOpenConnections, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.openConnections, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUse, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosed, prometheus.CounterValue, float64(stats.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxIdleTimeClosed, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeClosed, prometheus.CounterValue, float64(stats.MaxLifetimeClosed))
}
