package models

// AnalyticsSnapshot — агрегаты для бизнес-дашборда Grafana (см.
// internal/metrics/admin.Collector). Пересчитывается заново на каждый
// Prometheus-скрейп, не кэшируется — единственный потребитель дёргает его
// раз в scrape_interval, отдельное кэширование не нужно.
type AnalyticsSnapshot struct {
	TotalRevenue   float64
	TotalPurchases int64
	TotalUsers     int64
	BannedUsers    int64
	AdminUsers     int64
	TotalBalance   float64
	AvailableStock int64
	TopProducts    []RevenueByName
	TopCategories  []RevenueByName
}

// RevenueByName — одна строка топа (продукт или категория) по выручке.
type RevenueByName struct {
	Name      string
	Revenue   float64
	UnitsSold int64
}
