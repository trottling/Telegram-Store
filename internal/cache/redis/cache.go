package redis

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
)

// Cache — Redis-реализация domain/cache, domain/fsm.Store и
// domain/adminsession.Store сразу: один клиент, три непересекающихся
// пространства ключей.
type Cache struct {
	client *redis.Client
	log    *zap.SugaredLogger
}

func NewRedisCache(client *redis.Client, log *zap.SugaredLogger) *Cache {
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
		c.log.Errorw("cache: redis GET failed", "error", err, "key", key)
		return err
	}
	if err = json.Unmarshal(raw, dest); err != nil {
		// Битое значение удаляем сразу: иначе каждое чтение до истечения TTL
		// снова падает и снова пишет в лог, хотя чинится это одним DEL.
		c.log.Warnw("cache: stored value failed to unmarshal, dropping key", "error", err, "key", key)
		_ = c.delete(ctx, key)
		return domaincache.ErrMiss
	}
	return nil
}

func (c *Cache) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		c.log.Errorw("cache: failed to marshal value", "error", err, "key", key)
		return err
	}
	if err = c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		c.log.Warnw("cache: redis SET failed", "error", err, "key", key)
		return err
	}
	return nil
}

func (c *Cache) delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.log.Warnw("cache: redis DEL failed", "error", err, "keys", keys)
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
		c.log.Errorw("cache: redis GET failed", "error", err, "product_id", productID)
		return 0, err
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		c.log.Warnw("cache: stored count is not an int, dropping key", "error", err, "product_id", productID)
		_ = c.delete(ctx, productCountKey(productID))
		return 0, domaincache.ErrMiss
	}
	return count, nil
}

func (c *Cache) SetProductAvailableCount(ctx context.Context, productID int64, count int) error {
	if err := c.client.Set(ctx, productCountKey(productID), strconv.Itoa(count), productCountTTL).Err(); err != nil {
		c.log.Warnw("cache: redis SET failed", "error", err, "product_id", productID)
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
		c.log.Errorw("cache: redis GET failed (fsm state)", "error", err, "telegram_id", telegramID)
		return nil, err
	}

	var st domainfsm.State
	if err = json.Unmarshal(raw, &st); err != nil {
		c.log.Warnw("cache: stored fsm state failed to unmarshal, treating as absent", "error", err, "telegram_id", telegramID)
		return nil, domainfsm.ErrNotFound
	}
	return &st, nil
}

// ConsumeFSMState — GETDEL, чтобы два параллельных тапа по одной кнопке не
// прочитали одно состояние дважды (ср. ConsumeLoginCode).
func (c *Cache) ConsumeFSMState(ctx context.Context, telegramID int64) (*domainfsm.State, error) {
	raw, err := c.client.GetDel(ctx, stateKey(telegramID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domainfsm.ErrNotFound
	}
	if err != nil {
		c.log.Errorw("cache: redis GETDEL failed (fsm state)", "error", err, "telegram_id", telegramID)
		return nil, err
	}

	var st domainfsm.State
	if err = json.Unmarshal(raw, &st); err != nil {
		c.log.Warnw("cache: stored fsm state failed to unmarshal, treating as absent", "error", err, "telegram_id", telegramID)
		return nil, domainfsm.ErrNotFound
	}
	return &st, nil
}

func (c *Cache) SetFSMState(ctx context.Context, telegramID int64, st *domainfsm.State) error {
	raw, err := json.Marshal(st)
	if err != nil {
		c.log.Errorw("cache: failed to marshal fsm state", "error", err, "telegram_id", telegramID)
		return err
	}
	if err = c.client.Set(ctx, stateKey(telegramID), raw, stateTTL).Err(); err != nil {
		c.log.Errorw("cache: redis SET failed (fsm state)", "error", err, "telegram_id", telegramID)
		return err
	}
	return nil
}

func (c *Cache) ClearFSMState(ctx context.Context, telegramID int64) error {
	if err := c.client.Del(ctx, stateKey(telegramID)).Err(); err != nil {
		c.log.Warnw("cache: redis DEL failed (fsm state)", "error", err, "telegram_id", telegramID)
		return err
	}
	return nil
}

// сессии веб-панели

func (c *Cache) SetSession(ctx context.Context, sessionHash string, telegramID int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, adminSessionKey(sessionHash), strconv.FormatInt(telegramID, 10), ttl).Err(); err != nil {
		c.log.Warnw("cache: redis SET failed (admin session)", "error", err)
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
		c.log.Errorw("cache: redis GET failed (admin session)", "error", err)
		return 0, err
	}
	telegramID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.log.Warnw("cache: stored admin session value is not an int, treating as miss", "error", err)
		return 0, adminsession.ErrNotFound
	}
	return telegramID, nil
}

func (c *Cache) DeleteSession(ctx context.Context, sessionHash string) error {
	return c.delete(ctx, adminSessionKey(sessionHash))
}

// IncrExchangeAttempts — фиксированное окно: EXPIRE ставится только на первом
// INCR, иначе каждая новая попытка продлевала бы окно и оно никогда бы не
// истекло. Точность окна тут не важна — задача сбить перебор на порядки, а не
// отмерить ровный интервал.
func (c *Cache) IncrExchangeAttempts(ctx context.Context, key string, window time.Duration) (int64, error) {
	redisKey := adminExchangeAttemptsKey(key)

	attempt, err := c.client.Incr(ctx, redisKey).Result()
	if err != nil {
		c.log.Errorw("cache: redis INCR failed (exchange attempts)", "error", err)
		return 0, err
	}
	if attempt == 1 {
		if err = c.client.Expire(ctx, redisKey, window).Err(); err != nil {
			c.log.Warnw("cache: redis EXPIRE failed (exchange attempts)", "error", err)
		}
	}
	return attempt, nil
}

// TryAcquire — SETNX атомарен: в окне TTL только первый вызов вернёт true,
// остальные до истечения — false, без гонки между параллельными тапами
// одной кнопки.
func (c *Cache) TryAcquire(ctx context.Context, replenishmentID int64) (bool, error) {
	acquired, err := c.client.SetNX(ctx, replenishmentCheckCooldownKey(replenishmentID), 1, replenishmentCheckCooldownTTL).Result()
	if err != nil {
		c.log.Errorw("cache: redis SETNX failed (replenishment check cooldown)", "error", err, "replenishment_id", replenishmentID)
		return false, err
	}
	return acquired, nil
}
