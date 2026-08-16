package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trottling/TG-Store/internal/domain/models"
	"github.com/trottling/TG-Store/internal/domain/repository"
	"golang.org/x/sync/errgroup"
)

// defaultSeriesWindow — окно графика выручки по умолчанию, если период не задан.
const defaultSeriesWindow = 30 * 24 * time.Hour

const topListLimit = 10

type StatsSrv struct {
	statsRepo repository.StatsRepository
	log       *logrus.Logger
}

func NewStatsSrv(statsRepo repository.StatsRepository, log *logrus.Logger) *StatsSrv {
	return &StatsSrv{statsRepo: statsRepo, log: log}
}

// GetDashboard параллельно собирает пять запросов StatsRepository в один ответ.
func (s *StatsSrv) GetDashboard(ctx context.Context, from, to *time.Time) (*models.DashboardStats, error) {
	seriesFrom, seriesTo := from, to
	if seriesFrom == nil {
		t := time.Now().Add(-defaultSeriesWindow)
		seriesFrom = &t
	}
	if seriesTo == nil {
		t := time.Now()
		seriesTo = &t
	}

	var (
		overview  models.SalesOverview
		series    []models.RevenuePoint
		topProd   []models.ProductStat
		topCat    []models.CategoryStat
		userStats models.UserStats
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		o, err := s.statsRepo.GetSalesOverview(gctx, from, to)
		if err != nil {
			return err
		}
		overview = *o
		return nil
	})
	g.Go(func() (err error) {
		series, err = s.statsRepo.GetRevenueTimeSeries(gctx, *seriesFrom, *seriesTo)
		return err
	})
	g.Go(func() (err error) {
		topProd, err = s.statsRepo.GetTopProducts(gctx, from, to, topListLimit, models.StatsOrderByRevenue)
		return err
	})
	g.Go(func() (err error) {
		topCat, err = s.statsRepo.GetTopCategories(gctx, from, to, topListLimit, models.StatsOrderByRevenue)
		return err
	})
	g.Go(func() error {
		u, err := s.statsRepo.GetUserStats(gctx)
		if err != nil {
			return err
		}
		userStats = *u
		return nil
	})

	if err := g.Wait(); err != nil {
		s.log.WithError(err).Error("stats_service: get dashboard failed")
		return nil, err
	}

	return &models.DashboardStats{
		Overview:      overview,
		RevenueSeries: series,
		TopProducts:   topProd,
		TopCategories: topCat,
		Users:         userStats,
	}, nil
}
