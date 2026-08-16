package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// Cache — Redis-реализация domain/cache, domain/fsm.Store и
// domain/adminsession.Store сразу: один клиент, три непересекающихся
// пространства ключей.
type Cache struct {
	client *redis.Client
	log    *logrus.Logger
}

func NewRedisCache(client *redis.Client, log *logrus.Logger) *Cache {
	return &Cache{client: client, log: log}
}

// helpers

// getJSON не логирует обычный промах (redis.Nil) — это штатный путь.
func (c *Cache) getJSON(ctx context.Context, key string, dest any) error {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domaincache.ErrMiss
	}
	if err != nil {
		c.log.WithError(err).WithField("key", key).Error("cache: redis GET failed")
		return err
	}
	if err = json.Unmarshal(raw, dest); err != nil {
		c.log.WithError(err).WithField("key", key).Warn("cache: stored value failed to unmarshal, treating as miss")
		return domaincache.ErrMiss
	}
	return nil
}

func (c *Cache) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		c.log.WithError(err).WithField("key", key).Error("cache: failed to marshal value")
		return err
	}
	if err = c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		c.log.WithError(err).WithField("key", key).Warn("cache: redis SET failed")
		return err
	}
	return nil
}

func (c *Cache) delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.log.WithError(err).WithField("keys", keys).Warn("cache: redis DEL failed")
		return err
	}
	return nil
}

// пользователь

func (c *Cache) GetUser(ctx context.Context, telegramID int64) (*models.User, error) {
	var user models.User
	if err := c.getJSON(ctx, userKey(telegramID), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Cache) SetUser(ctx context.Context, user *models.User) error {
	return c.setJSON(ctx, userKey(user.TelegramID), user, userTTL)
}

func (c *Cache) InvalidateUser(ctx context.Context, telegramID int64) error {
	return c.delete(ctx, userKey(telegramID))
}

// товары

func (c *Cache) GetActiveProducts(ctx context.Context) ([]models.Product, error) {
	var products []models.Product
	if err := c.getJSON(ctx, activeProductsKey(), &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (c *Cache) SetActiveProducts(ctx context.Context, products []models.Product) error {
	return c.setJSON(ctx, activeProductsKey(), products, activeProductsTTL)
}

func (c *Cache) InvalidateActiveProducts(ctx context.Context) error {
	return c.delete(ctx, activeProductsKey())
}

func (c *Cache) GetProduct(ctx context.Context, productID int64) (*models.Product, error) {
	var product models.Product
	if err := c.getJSON(ctx, productKey(productID), &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (c *Cache) SetProduct(ctx context.Context, product *models.Product) error {
	return c.setJSON(ctx, productKey(product.ID), product, productTTL)
}

func (c *Cache) InvalidateProduct(ctx context.Context, productID int64) error {
	return c.delete(ctx, productKey(productID))
}

func (c *Cache) GetProductAvailableCount(ctx context.Context, productID int64) (int, error) {
	raw, err := c.client.Get(ctx, productCountKey(productID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, domaincache.ErrMiss
	}
	if err != nil {
		c.log.WithError(err).WithField("product_id", productID).Error("cache: redis GET failed")
		return 0, err
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		c.log.WithError(err).WithField("product_id", productID).Warn("cache: stored count is not an int, treating as miss")
		return 0, domaincache.ErrMiss
	}
	return count, nil
}

func (c *Cache) SetProductAvailableCount(ctx context.Context, productID int64, count int) error {
	if err := c.client.Set(ctx, productCountKey(productID), strconv.Itoa(count), productCountTTL).Err(); err != nil {
		c.log.WithError(err).WithField("product_id", productID).Warn("cache: redis SET failed")
		return err
	}
	return nil
}

func (c *Cache) InvalidateProductAvailableCount(ctx context.Context, productID int64) error {
	return c.delete(ctx, productCountKey(productID))
}

// категории

func (c *Cache) GetCategoryChildren(ctx context.Context, parentID *int64) ([]models.Category, error) {
	var categories []models.Category
	if err := c.getJSON(ctx, categoryChildrenKey(parentID), &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (c *Cache) SetCategoryChildren(ctx context.Context, parentID *int64, categories []models.Category) error {
	return c.setJSON(ctx, categoryChildrenKey(parentID), categories, categoryChildrenTTL)
}

func (c *Cache) InvalidateCategoryChildren(ctx context.Context, parentID *int64) error {
	return c.delete(ctx, categoryChildrenKey(parentID))
}

// настройки бота

func (c *Cache) GetSettings(ctx context.Context) (*models.Settings, error) {
	var settings models.Settings
	if err := c.getJSON(ctx, settingsKey(), &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (c *Cache) SetSettings(ctx context.Context, settings *models.Settings) error {
	return c.setJSON(ctx, settingsKey(), settings, settingsTTL)
}

func (c *Cache) InvalidateSettings(ctx context.Context) error {
	return c.delete(ctx, settingsKey())
}

// состояние FSM

func (c *Cache) GetFSMState(ctx context.Context, telegramID int64) (*domainfsm.State, error) {
	raw, err := c.client.Get(ctx, stateKey(telegramID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domainfsm.ErrNotFound
	}
	if err != nil {
		c.log.WithError(err).WithField("telegram_id", telegramID).Error("cache: redis GET failed (fsm state)")
		return nil, err
	}

	var st domainfsm.State
	if err = json.Unmarshal(raw, &st); err != nil {
		c.log.WithError(err).WithField("telegram_id", telegramID).Warn("cache: stored fsm state failed to unmarshal, treating as absent")
		return nil, domainfsm.ErrNotFound
	}
	return &st, nil
}

func (c *Cache) SetFSMState(ctx context.Context, telegramID int64, st *domainfsm.State) error {
	raw, err := json.Marshal(st)
	if err != nil {
		c.log.WithError(err).WithField("telegram_id", telegramID).Error("cache: failed to marshal fsm state")
		return err
	}
	if err = c.client.Set(ctx, stateKey(telegramID), raw, stateTTL).Err(); err != nil {
		c.log.WithError(err).WithField("telegram_id", telegramID).Error("cache: redis SET failed (fsm state)")
		return err
	}
	return nil
}

func (c *Cache) ClearFSMState(ctx context.Context, telegramID int64) error {
	if err := c.client.Del(ctx, stateKey(telegramID)).Err(); err != nil {
		c.log.WithError(err).WithField("telegram_id", telegramID).Warn("cache: redis DEL failed (fsm state)")
		return err
	}
	return nil
}

// логин-коды и сессии веб-панели

func (c *Cache) SetLoginCode(ctx context.Context, codeHash string, telegramID int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, adminLoginCodeKey(codeHash), strconv.FormatInt(telegramID, 10), ttl).Err(); err != nil {
		c.log.WithError(err).Warn("cache: redis SET failed (admin login code)")
		return err
	}
	return nil
}

// ConsumeLoginCode использует GETDEL — атомарно, код нельзя обменять дважды.
func (c *Cache) ConsumeLoginCode(ctx context.Context, codeHash string) (int64, error) {
	raw, err := c.client.GetDel(ctx, adminLoginCodeKey(codeHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, adminsession.ErrNotFound
	}
	if err != nil {
		c.log.WithError(err).Error("cache: redis GETDEL failed (admin login code)")
		return 0, err
	}
	telegramID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.log.WithError(err).Warn("cache: stored admin login code value is not an int, treating as miss")
		return 0, adminsession.ErrNotFound
	}
	return telegramID, nil
}

func (c *Cache) SetSession(ctx context.Context, sessionHash string, telegramID int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, adminSessionKey(sessionHash), strconv.FormatInt(telegramID, 10), ttl).Err(); err != nil {
		c.log.WithError(err).Warn("cache: redis SET failed (admin session)")
		return err
	}
	return nil
}

func (c *Cache) GetSession(ctx context.Context, sessionHash string) (int64, error) {
	raw, err := c.client.Get(ctx, adminSessionKey(sessionHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, adminsession.ErrNotFound
	}
	if err != nil {
		c.log.WithError(err).Error("cache: redis GET failed (admin session)")
		return 0, err
	}
	telegramID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.log.WithError(err).Warn("cache: stored admin session value is not an int, treating as miss")
		return 0, adminsession.ErrNotFound
	}
	return telegramID, nil
}

func (c *Cache) DeleteSession(ctx context.Context, sessionHash string) error {
	return c.delete(ctx, adminSessionKey(sessionHash))
}
