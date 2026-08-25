package postgres

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AnalyticsRepo — агрегаты для бизнес-дашборда Grafana через GORM query
// builder (classic *gorm.DB, не gorm.G[T] — GROUP BY туда не укладывается,
// тот же выбор, что в PurchaseRepo.ListBatchesByUserID).
type AnalyticsRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewAnalyticsRepo(db *gorm.DB, log *zap.SugaredLogger) *AnalyticsRepo {
	return &AnalyticsRepo{db: db, log: log}
}

func (r *AnalyticsRepo) GetSnapshot(ctx context.Context, topN int) (*models.AnalyticsSnapshot, error) {
	snap := &models.AnalyticsSnapshot{}

	var sales struct {
		TotalRevenue   models.Money
		TotalPurchases int64
	}
	if err := r.db.WithContext(ctx).Model(&models.Purchase{}).
		Select("COALESCE(SUM(amount), 0) AS total_revenue, COUNT(*) AS total_purchases").
		Where("status = ?", models.PurchaseStatusCompleted).
		Scan(&sales).Error; err != nil {
		r.log.Errorw("analytics_repo: get sales snapshot failed", "error", err)
		return nil, err
	}
	snap.TotalRevenue = sales.TotalRevenue
	snap.TotalPurchases = sales.TotalPurchases

	var users struct {
		TotalUsers   int64
		BannedUsers  int64
		AdminUsers   int64
		TotalBalance models.Money
	}
	if err := r.db.WithContext(ctx).Model(&userRecord{}).
		Select("COUNT(*) AS total_users, "+
			"COUNT(*) FILTER (WHERE role = ?) AS banned_users, "+
			"COUNT(*) FILTER (WHERE role IN (?, ?)) AS admin_users, "+
			"COALESCE(SUM(balance), 0) AS total_balance",
			models.RoleBanned, models.RoleAdmin, models.RoleRootAdmin).
		Scan(&users).Error; err != nil {
		r.log.Errorw("analytics_repo: get user snapshot failed", "error", err)
		return nil, err
	}
	snap.TotalUsers = users.TotalUsers
	snap.BannedUsers = users.BannedUsers
	snap.AdminUsers = users.AdminUsers
	snap.TotalBalance = users.TotalBalance

	var availableStock int64
	if err := r.db.WithContext(ctx).Model(&models.ProductItem{}).
		Where("is_sold = false").
		Count(&availableStock).Error; err != nil {
		r.log.Errorw("analytics_repo: get available stock failed", "error", err)
		return nil, err
	}
	snap.AvailableStock = availableStock

	if err := r.db.WithContext(ctx).Model(&models.Purchase{}).
		Select("pr.name AS name, SUM(purchases.amount) AS revenue, COUNT(*) AS units_sold").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Where("purchases.status = ?", models.PurchaseStatusCompleted).
		Group("pr.id, pr.name").
		Order("revenue DESC").
		Limit(topN).
		Scan(&snap.TopProducts).Error; err != nil {
		r.log.Errorw("analytics_repo: get top products failed", "error", err)
		return nil, err
	}

	if err := r.db.WithContext(ctx).Model(&models.Purchase{}).
		Select("COALESCE(c.name, 'Uncategorized') AS name, SUM(purchases.amount) AS revenue, COUNT(*) AS units_sold").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Joins("LEFT JOIN categories c ON c.id = pr.category_id").
		Where("purchases.status = ?", models.PurchaseStatusCompleted).
		Group("c.id, c.name").
		Order("revenue DESC").
		Limit(topN).
		Scan(&snap.TopCategories).Error; err != nil {
		r.log.Errorw("analytics_repo: get top categories failed", "error", err)
		return nil, err
	}

	return snap, nil
}
