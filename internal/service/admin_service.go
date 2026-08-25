package service

import (
	"context"
	"encoding/json/v2"
	"strconv"

	"github.com/shopspring/decimal"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	adminmetrics "github.com/trottling/Telegram-Store/internal/metrics/admin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type AdminSrv struct {
	userRepo      repository.UserRepository
	productRepo   repository.ProductRepository
	categoryRepo  repository.CategoryRepository
	purchaseRepo  repository.PurchaseRepository
	adminLogRepo  repository.AdminLogRepository
	settingsRepo  repository.SettingsRepository
	cache         MultiCache
	settingsCache domaincache.SettingsCache
	log           *zap.SugaredLogger
}

func NewAdminSrv(
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	purchaseRepo repository.PurchaseRepository,
	adminLogRepo repository.AdminLogRepository,
	settingsRepo repository.SettingsRepository,
	cache MultiCache,
	settingsCache domaincache.SettingsCache,
	log *zap.SugaredLogger,
) *AdminSrv {
	return &AdminSrv{
		userRepo:      userRepo,
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		purchaseRepo:  purchaseRepo,
		adminLogRepo:  adminLogRepo,
		settingsRepo:  settingsRepo,
		cache:         cache,
		settingsCache: settingsCache,
		log:           log,
	}
}

func adminLogDetails(v any) datatypes.JSON {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return raw
}

// logAction — targetID уже отформатирован вызывающим в строку: TargetID
// полиморфен (то Telegram ID, то ProductID/CategoryID, то Settings.ID), у
// него нет единого типа (см. doc-комментарий models.AdminLog).
func (s *AdminSrv) logAction(ctx context.Context, adminID models.TelegramID, action string, targetID *string, details any) {
	_ = s.adminLogRepo.Create(ctx, &models.AdminLog{
		AdminID:  adminID,
		Action:   action,
		TargetID: targetID,
		Details:  adminLogDetails(details),
	})
	adminmetrics.ActionsTotal.WithLabelValues(action).Inc()
	s.log.Infow("admin_service: action performed", "admin_id", adminID, "action", action, "target_id", *targetID)
}

// strPtr — targetID в logAction принимает *string; большинство вызовов
// форматируют значение прямо на месте (targetTelegramID.String() и т.п.).
func strPtr(s string) *string { return &s }

func (s *AdminSrv) AddBalance(ctx context.Context, adminID, targetTelegramID models.TelegramID, amount decimal.Decimal) error {
	if amount.IsZero() {
		return domainerrors.ErrInvalidAmount
	}

	if _, err := s.userRepo.GetByID(ctx, targetTelegramID); err != nil {
		return err
	}

	if err := s.userRepo.UpdateBalance(ctx, targetTelegramID, amount); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "balance_add", strPtr(targetTelegramID.String()), map[string]any{"amount": amount})
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// BanUser банит через User.Ban — тот сам отказывает на root admin и на
// самого себя (иначе некому будет вернуть права обратно) и снимает права
// админа заодно (роль одна на всех).
func (s *AdminSrv) BanUser(ctx context.Context, adminID, targetTelegramID models.TelegramID) error {
	actor, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if err = target.Ban(actor); err != nil {
		return err
	}

	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "ban", strPtr(targetTelegramID.String()), nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// UnbanUser всегда возвращает роль User, а не ту, что была до бана —
// повторно выдать права нужно через MakeAdmin.
func (s *AdminSrv) UnbanUser(ctx context.Context, adminID, targetTelegramID models.TelegramID) error {
	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if !target.IsBanned() {
		return nil
	}

	target.Unban()
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "unban", strPtr(targetTelegramID.String()), nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// MakeAdmin выдаёт права админа через User.Promote — тот сам отказывает,
// если actor не root admin (иначе цепочка promote была бы неконтролируемой).
func (s *AdminSrv) MakeAdmin(ctx context.Context, adminID, targetTelegramID models.TelegramID) error {
	actingAdmin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if err = target.Promote(actingAdmin); err != nil {
		return err
	}

	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "make_admin", strPtr(targetTelegramID.String()), nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// RevokeAdmin снимает права админа через User.Demote — тот сам отказывает на
// root admin и на самого себя.
func (s *AdminSrv) RevokeAdmin(ctx context.Context, adminID, targetTelegramID models.TelegramID) error {
	actor, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if err = target.Demote(actor); err != nil {
		return err
	}

	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "revoke_admin", strPtr(targetTelegramID.String()), nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// SetReferralsEnabled вкл/выкл начисления targetTelegramID как рефереру —
// не трогает его собственный ReferrerID, только доступность его самого как
// приглашающего (см. PurchaseSrv.creditReferral).
func (s *AdminSrv) SetReferralsEnabled(ctx context.Context, adminID, targetTelegramID models.TelegramID, enabled bool) error {
	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if target.ReferralsEnabled == enabled {
		return nil
	}

	target.ReferralsEnabled = enabled
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	action := "referral_disable"
	if enabled {
		action = "referral_enable"
	}
	s.logAction(ctx, adminID, action, strPtr(targetTelegramID.String()), nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

func (s *AdminSrv) CreateProduct(ctx context.Context, adminID models.TelegramID, categoryID *models.CategoryID, name, description string, price models.Money) (*models.Product, error) {
	if name == "" || price.IsZero() {
		return nil, domainerrors.ErrInvalidProductInput
	}
	if categoryID != nil {
		if _, err := s.categoryRepo.GetByID(ctx, *categoryID); err != nil {
			return nil, err
		}
	}

	product := &models.Product{CategoryID: categoryID, Name: name, Description: description, Price: price, IsActive: true}
	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	s.logAction(ctx, adminID, "product_create", strPtr(product.ID.String()), nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	return product, nil
}

func (s *AdminSrv) UpdateProduct(ctx context.Context, adminID models.TelegramID, productID models.ProductID, categoryID *models.CategoryID, name, description string, price models.Money, isActive bool) (*models.Product, error) {
	if name == "" || price.IsZero() {
		return nil, domainerrors.ErrInvalidProductInput
	}
	if categoryID != nil {
		if _, err := s.categoryRepo.GetByID(ctx, *categoryID); err != nil {
			return nil, err
		}
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	oldCategoryID := product.CategoryID

	product.CategoryID = categoryID
	product.Name = name
	product.Description = description
	product.Price = price
	product.IsActive = isActive
	if err = s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	s.logAction(ctx, adminID, "product_update", strPtr(product.ID.String()), nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	_ = s.cache.InvalidateProduct(ctx, productID)
	// IsActive/смена категории влияют на остаток старой и новой категории вверх по дереву.
	if oldCategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *oldCategoryID)
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *oldCategoryID)
	}
	if categoryID != nil && (oldCategoryID == nil || *categoryID != *oldCategoryID) {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *categoryID)
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *categoryID)
	}
	return product, nil
}

func (s *AdminSrv) DeleteProduct(ctx context.Context, adminID models.TelegramID, productID models.ProductID) error {
	// Получаем товар заранее, чтобы знать категорию для инвалидации кэша.
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return err
	}

	// Не удаляем товар с историей покупок — вместо этого деактивация (IsActive).
	purchaseCount, err := s.purchaseRepo.CountByProductID(ctx, productID)
	if err != nil {
		return err
	}
	if purchaseCount > 0 {
		return domainerrors.ErrProductHasPurchases
	}

	if err = s.productRepo.Delete(ctx, productID); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "product_delete", strPtr(productID.String()), nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	_ = s.cache.InvalidateProduct(ctx, productID)
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	if product.CategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *product.CategoryID)
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *product.CategoryID)
	}
	return nil
}

func (s *AdminSrv) AddProductItems(ctx context.Context, adminID models.TelegramID, productID models.ProductID, contents []string) error {
	if len(contents) == 0 {
		return domainerrors.ErrNoItemsProvided
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return err
	}

	if err = s.productRepo.AddItems(ctx, productID, contents); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "product_add_items", strPtr(productID.String()), map[string]any{"count": len(contents)})
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	// Товар с новым остатком может снова попасть в листинг, категория тоже могла быть скрыта при нуле.
	_ = s.cache.InvalidateActiveProducts(ctx)
	if product.CategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *product.CategoryID)
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *product.CategoryID)
	}
	return nil
}

func (s *AdminSrv) CreateCategory(ctx context.Context, adminID models.TelegramID, parentID *models.CategoryID, name, description string) (*models.Category, error) {
	if name == "" {
		return nil, domainerrors.ErrInvalidProductInput
	}
	if parentID != nil {
		if _, err := s.categoryRepo.GetByID(ctx, *parentID); err != nil {
			return nil, err
		}
	}

	category := &models.Category{ParentID: parentID, Name: name, Description: description}
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	s.logAction(ctx, adminID, "category_create", strPtr(category.ID.String()), nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, parentID)
	return category, nil
}

func (s *AdminSrv) UpdateCategory(ctx context.Context, adminID models.TelegramID, categoryID models.CategoryID, name, description string, parentID *models.CategoryID) (*models.Category, error) {
	if name == "" {
		return nil, domainerrors.ErrInvalidProductInput
	}
	if parentID != nil {
		if *parentID == categoryID {
			return nil, domainerrors.ErrInvalidProductInput
		}
		if _, err := s.categoryRepo.GetByID(ctx, *parentID); err != nil {
			return nil, err
		}
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	oldParentID := category.ParentID
	category.Name = name
	category.Description = description
	category.ParentID = parentID
	if err = s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	s.logAction(ctx, adminID, "category_update", strPtr(category.ID.String()), nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, oldParentID)
	_ = s.cache.InvalidateCategoryChildren(ctx, parentID)
	// Перенос между родителями не меняет HasStock самой категории, только то,
	// чей агрегат её учитывает — пересчитываем у старого и нового родителя,
	// не у categoryID.
	if oldParentID != nil {
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *oldParentID)
	}
	if parentID != nil && (oldParentID == nil || *parentID != *oldParentID) {
		recomputeCategoryStockChain(ctx, s.categoryRepo, s.log, *parentID)
	}
	return category, nil
}

func (s *AdminSrv) DeleteCategory(ctx context.Context, adminID models.TelegramID, categoryID models.CategoryID) error {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}

	// Не удаляем непустую категорию — сперва нужно перенести/удалить потомков и товары.
	childCount, err := s.categoryRepo.CountChildren(ctx, categoryID)
	if err != nil {
		return err
	}
	productCount, err := s.productRepo.CountByCategoryID(ctx, categoryID)
	if err != nil {
		return err
	}
	if childCount > 0 || productCount > 0 {
		return domainerrors.ErrCategoryNotEmpty
	}

	if err = s.categoryRepo.Delete(ctx, categoryID); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "category_delete", strPtr(categoryID.String()), nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, category.ParentID)
	return nil
}

// UpdateSettings перезаписывает единственную строку настроек бота целиком.
func (s *AdminSrv) UpdateSettings(ctx context.Context, adminID models.TelegramID, settings *models.Settings) (*models.Settings, error) {
	if settings.SupportUsername == "" {
		return nil, domainerrors.ErrInvalidSettingsInput
	}
	if settings.Referral.Percent < 0 || settings.Referral.Percent > 100 {
		return nil, domainerrors.ErrInvalidSettingsInput
	}

	settings.ID = models.SettingsID
	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		return nil, err
	}

	s.logAction(ctx, adminID, "settings_update", strPtr(strconv.FormatInt(settings.ID, 10)), nil)
	logInvalidation(s.log, s.settingsCache.InvalidateSettings(ctx), "settings", models.SettingsID)
	return settings, nil
}

func (s *AdminSrv) GetLogs(ctx context.Context, adminID models.TelegramID, offset, limit int) ([]models.AdminLog, error) {
	return s.adminLogRepo.ListByAdmin(ctx, adminID, offset, limit)
}

// ListLogs/CountLogs — журнал по всем админам.
func (s *AdminSrv) ListLogs(ctx context.Context, adminID *models.TelegramID, offset, limit int) ([]models.AdminLog, error) {
	return s.adminLogRepo.ListAll(ctx, adminID, offset, limit)
}

func (s *AdminSrv) CountLogs(ctx context.Context, adminID *models.TelegramID) (int64, error) {
	return s.adminLogRepo.CountAll(ctx, adminID)
}
