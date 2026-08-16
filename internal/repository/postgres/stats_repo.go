package postgres

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"gorm.io/gorm"
)

// StatsRepo — агрегаты для экрана статистики через GORM query builder, не .Raw().
type StatsRepo struct {
	db  *gorm.DB
	log *logrus.Logger
}

func NewStatsRepo(db *gorm.DB, log *logrus.Logger) *StatsRepo {
	return &StatsRepo{db: db, log: log}
}

func (r *StatsRepo) GetSalesOverview(ctx context.Context, from, to *time.Time) (*models.SalesOverview, error) {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("COUNT(*) AS total_purchases, COALESCE(SUM(amount), 0) AS total_revenue").
		Where("status = ?", models.PurchaseStatusCompleted)
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}

	var out models.SalesOverview
	if err := q.Scan(&out).Error; err != nil {
		r.log.WithError(err).Error("stats_repo: get sales overview failed")
		return nil, err
	}
	return &out, nil
}

func (r *StatsRepo) GetRevenueTimeSeries(ctx context.Context, from, to time.Time) ([]models.RevenuePoint, error) {
	var points []models.RevenuePoint
	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("date_trunc('day', created_at) AS date, COALESCE(SUM(amount), 0) AS revenue, COUNT(*) AS count").
		Where("status = ? AND created_at BETWEEN ? AND ?", models.PurchaseStatusCompleted, from, to).
		Group("date_trunc('day', created_at)").
		Order("date_trunc('day', created_at)").
		Scan(&points).Error
	if err != nil {
		r.log.WithError(err).Error("stats_repo: get revenue time series failed")
	}
	return points, err
}

// statsOrderColumn — enum в SQL-идентификатор, не сырой параметр запроса напрямую.
func statsOrderColumn(orderBy models.StatsOrderBy) string {
	if orderBy == models.StatsOrderByUnits {
		return "units_sold"
	}
	return "revenue"
}

func (r *StatsRepo) GetTopProducts(ctx context.Context, from, to *time.Time, limit int, orderBy models.StatsOrderBy) ([]models.ProductStat, error) {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("pr.id AS product_id, pr.name AS product_name, COUNT(*) AS units_sold, SUM(purchases.amount) AS revenue").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Where("purchases.status = ?", models.PurchaseStatusCompleted)
	if from != nil {
		q = q.Where("purchases.created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("purchases.created_at <= ?", *to)
	}

	var out []models.ProductStat
	err := q.Group("pr.id, pr.name").
		Order(statsOrderColumn(orderBy) + " DESC").
		Limit(limit).
		Scan(&out).Error
	if err != nil {
		r.log.WithError(err).Error("stats_repo: get top products failed")
	}
	return out, err
}

func (r *StatsRepo) GetTopCategories(ctx context.Context, from, to *time.Time, limit int, orderBy models.StatsOrderBy) ([]models.CategoryStat, error) {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("c.id AS category_id, COALESCE(c.name, 'Uncategorized') AS category_name, COUNT(*) AS units_sold, SUM(purchases.amount) AS revenue").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Joins("LEFT JOIN categories c ON c.id = pr.category_id").
		Where("purchases.status = ?", models.PurchaseStatusCompleted)
	if from != nil {
		q = q.Where("purchases.created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("purchases.created_at <= ?", *to)
	}

	var out []models.CategoryStat
	err := q.Group("c.id, c.name").
		Order(statsOrderColumn(orderBy) + " DESC").
		Limit(limit).
		Scan(&out).Error
	if err != nil {
		r.log.WithError(err).Error("stats_repo: get top categories failed")
	}
	return out, err
}

func (r *StatsRepo) GetUserStats(ctx context.Context) (*models.UserStats, error) {
	var out models.UserStats
	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.User{}).
		Select("COUNT(*) AS total_users, COUNT(*) FILTER (WHERE role = 'banned') AS banned_users, COUNT(*) FILTER (WHERE role IN ('admin', 'root_admin')) AS admin_users, COALESCE(SUM(balance), 0) AS total_balance").
		Where("deleted_at IS NULL").
		Scan(&out).Error
	if err != nil {
		r.log.WithError(err).Error("stats_repo: get user stats failed")
		return nil, err
	}
	return &out, nil
}
