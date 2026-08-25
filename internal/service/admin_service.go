package service

import (
	"context"
	"encoding/json/v2"

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

func (s *AdminSrv) logAction(ctx context.Context, adminID int64, action string, targetID *int64, details any) {
	_ = s.adminLogRepo.Create(ctx, &models.AdminLog{
		AdminID:  adminID,
		Action:   action,
		TargetID: targetID,
		Details:  adminLogDetails(details),
	})
	adminmetrics.ActionsTotal.WithLabelValues(action).Inc()
	s.log.Infow("admin_service: action performed", "admin_id", adminID, "action", action, "target_id", *targetID)
}

func (s *AdminSrv) AddBalance(ctx context.Context, adminID, targetTelegramID int64, amount float64) error {
	if amount == 0 {
		return domainerrors.ErrInvalidAmount
	}

	if _, err := s.userRepo.GetByID(ctx, targetTelegramID); err != nil {
		return err
	}

	if err := s.userRepo.UpdateBalance(ctx, targetTelegramID, amount); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "balance_add", &targetTelegramID, map[string]any{"amount": amount})
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// BanUser ставит роль Banned напрямую (роль одна — банит и снимает права
// админа заодно). Отказывает на root admin и на самого себя — иначе некому
// будет вернуть права обратно.
func (s *AdminSrv) BanUser(ctx context.Context, adminID, targetTelegramID int64) error {
	if targetTelegramID == adminID {
		return domainerrors.ErrCannotBanSelf
	}

	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if target.IsRootAdmin() {
		return domainerrors.ErrCannotBanRootAdmin
	}
	if target.IsBanned() {
		return nil
	}

	target.Role = models.RoleBanned
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "ban", &targetTelegramID, nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// UnbanUser всегда возвращает роль User, а не ту, что была до бана —
// повторно выдать права нужно через MakeAdmin.
func (s *AdminSrv) UnbanUser(ctx context.Context, adminID, targetTelegramID int64) error {
	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if !target.IsBanned() {
		return nil
	}

	target.Role = models.RoleUser
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "unban", &targetTelegramID, nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// MakeAdmin выдаёт права админа — только для root admin, иначе цепочка
// promote была бы неконтролируемой.
func (s *AdminSrv) MakeAdmin(ctx context.Context, adminID, targetTelegramID int64) error {
	actingAdmin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	if !actingAdmin.IsRootAdmin() {
		return domainerrors.ErrOnlyRootAdminCanPromote
	}

	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if target.IsAdmin() {
		return domainerrors.ErrAlreadyAdmin
	}

	target.Role = models.RoleAdmin
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "make_admin", &targetTelegramID, nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// RevokeAdmin снимает права админа. Нельзя снять с root admin или с себя.
func (s *AdminSrv) RevokeAdmin(ctx context.Context, adminID, targetTelegramID int64) error {
	if targetTelegramID == adminID {
		return domainerrors.ErrCannotRevokeSelf
	}

	target, err := s.userRepo.GetByID(ctx, targetTelegramID)
	if err != nil {
		return err
	}
	if target.IsRootAdmin() {
		return domainerrors.ErrCannotRevokeRootAdmin
	}
	if !target.IsAdmin() {
		return domainerrors.ErrNotAdmin
	}

	target.Role = models.RoleUser
	if err = s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	s.logAction(ctx, adminID, "revoke_admin", &targetTelegramID, nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

// SetReferralsEnabled вкл/выкл начисления targetTelegramID как рефереру —
// не трогает его собственный ReferrerID, только доступность его самого как
// приглашающего (см. PurchaseSrv.creditReferral).
func (s *AdminSrv) SetReferralsEnabled(ctx context.Context, adminID, targetTelegramID int64, enabled bool) error {
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
	s.logAction(ctx, adminID, action, &targetTelegramID, nil)
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, targetTelegramID), "user", targetTelegramID)
	return nil
}

func (s *AdminSrv) CreateProduct(ctx context.Context, adminID int64, categoryID *int64, name, description string, price float64) (*models.Product, error) {
	if name == "" || price <= 0 {
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

	s.logAction(ctx, adminID, "product_create", &product.ID, nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	return product, nil
}

func (s *AdminSrv) UpdateProduct(ctx context.Context, adminID int64, productID int64, categoryID *int64, name, description string, price float64, isActive bool) (*models.Product, error) {
	if name == "" || price <= 0 {
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

	s.logAction(ctx, adminID, "product_update", &product.ID, nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	_ = s.cache.InvalidateProduct(ctx, productID)
	// IsActive/смена категории влияют на остаток старой и новой категории вверх по дереву.
	if oldCategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *oldCategoryID)
	}
	if categoryID != nil && (oldCategoryID == nil || *categoryID != *oldCategoryID) {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *categoryID)
	}
	return product, nil
}

func (s *AdminSrv) DeleteProduct(ctx context.Context, adminID int64, productID int64) error {
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

	s.logAction(ctx, adminID, "product_delete", &productID, nil)
	_ = s.cache.InvalidateActiveProducts(ctx)
	_ = s.cache.InvalidateProduct(ctx, productID)
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	if product.CategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *product.CategoryID)
	}
	return nil
}

func (s *AdminSrv) AddProductItems(ctx context.Context, adminID int64, productID int64, contents []string) error {
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

	s.logAction(ctx, adminID, "product_add_items", &productID, map[string]any{"count": len(contents)})
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	// Товар с новым остатком может снова попасть в листинг, категория тоже могла быть скрыта при нуле.
	_ = s.cache.InvalidateActiveProducts(ctx)
	if product.CategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *product.CategoryID)
	}
	return nil
}

func (s *AdminSrv) CreateCategory(ctx context.Context, adminID int64, parentID *int64, name, description string) (*models.Category, error) {
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

	s.logAction(ctx, adminID, "category_create", &category.ID, nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, parentID)
	return category, nil
}

func (s *AdminSrv) UpdateCategory(ctx context.Context, adminID int64, categoryID int64, name, description string, parentID *int64) (*models.Category, error) {
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

	s.logAction(ctx, adminID, "category_update", &category.ID, nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, oldParentID)
	_ = s.cache.InvalidateCategoryChildren(ctx, parentID)
	return category, nil
}

func (s *AdminSrv) DeleteCategory(ctx context.Context, adminID int64, categoryID int64) error {
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

	s.logAction(ctx, adminID, "category_delete", &categoryID, nil)
	_ = s.cache.InvalidateCategoryChildren(ctx, category.ParentID)
	return nil
}

// UpdateSettings перезаписывает единственную строку настроек бота целиком.
func (s *AdminSrv) UpdateSettings(ctx context.Context, adminID int64, settings *models.Settings) (*models.Settings, error) {
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

	s.logAction(ctx, adminID, "settings_update", &settings.ID, nil)
	logInvalidation(s.log, s.settingsCache.InvalidateSettings(ctx), "settings", models.SettingsID)
	return settings, nil
}

func (s *AdminSrv) GetLogs(ctx context.Context, adminID int64, offset, limit int) ([]models.AdminLog, error) {
	return s.adminLogRepo.ListByAdmin(ctx, adminID, offset, limit)
}

// ListLogs/CountLogs — журнал по всем админам.
func (s *AdminSrv) ListLogs(ctx context.Context, adminID *int64, offset, limit int) ([]models.AdminLog, error) {
	return s.adminLogRepo.ListAll(ctx, adminID, offset, limit)
}

func (s *AdminSrv) CountLogs(ctx context.Context, adminID *int64) (int64, error) {
	return s.adminLogRepo.CountAll(ctx, adminID)
}
