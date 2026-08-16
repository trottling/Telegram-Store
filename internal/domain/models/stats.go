package models

import "time"

// StatsOrderBy — сортировка для топ-товаров/топ-категорий.
type StatsOrderBy string

const (
	StatsOrderByRevenue StatsOrderBy = "revenue"
	StatsOrderByUnits   StatsOrderBy = "units"
)

// SalesOverview — общие цифры на экране статистики, только завершённые покупки.
type SalesOverview struct {
	TotalRevenue   float64 `json:"total_revenue"`
	TotalPurchases int64   `json:"total_purchases"`
}

// RevenuePoint — одна точка графика выручки (день).
type RevenuePoint struct {
	Date    time.Time `json:"date"`
	Revenue float64   `json:"revenue"`
	Count   int64     `json:"count"`
}

type ProductStat struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitsSold   int64   `json:"units_sold"`
	Revenue     float64 `json:"revenue"`
}

// CategoryStat — CategoryID == nil для товаров без категории.
type CategoryStat struct {
	CategoryID   *int64  `json:"category_id,omitempty"`
	CategoryName string  `json:"category_name"`
	UnitsSold    int64   `json:"units_sold"`
	Revenue      float64 `json:"revenue"`
}

type UserStats struct {
	TotalUsers   int64   `json:"total_users"`
	BannedUsers  int64   `json:"banned_users"`
	AdminUsers   int64   `json:"admin_users"`
	TotalBalance float64 `json:"total_balance"`
}

// DashboardStats — ответ GET /api/stats/dashboard.
type DashboardStats struct {
	Overview      SalesOverview  `json:"overview"`
	RevenueSeries []RevenuePoint `json:"revenue_series"`
	TopProducts   []ProductStat  `json:"top_products"`
	TopCategories []CategoryStat `json:"top_categories"`
	Users         UserStats      `json:"users"`
}
