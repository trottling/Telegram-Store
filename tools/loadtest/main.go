// Package main — синтетическая нагрузка на code path одного Telegram-апдейта,
// без самого Telegram: бьём напрямую в тот же internal/service/internal/repository
// стек, что использует bot/middleware, против настоящих Postgres/Redis
// (docker-compose.debug.yml). Идти через настоящий bot.Bot нельзя без сети —
// bot.New делает реальный GetMe к api.telegram.org — а измеряем мы всё равно
// стоимость нашей инфраструктуры, не сеть Telegram.
//
// Сценарий banprofile воспроизводит Tier 1 из плана оптимизации: то, что
// сейчас происходит на КАЖДОМ апдейте — bot/middleware.BanCheck
// (RefreshProfile, свежий поход в Postgres ради мгновенного бана) и вызов
// хендлера (GetProfile) — и проверяет, что второй поход действительно
// исчезает благодаря domainservice.WithUser/UserFromContext.
//
// Использование:
//
//	docker compose -f docker-compose.debug.yml up -d db redis
//	go run ./tools/loadtest -scenario banprofile -users 500 -concurrency 15 -duration 20s
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	svc "github.com/trottling/Telegram-Store/internal/service"
)

// loadtestUserBase — диапазон TelegramID для синтетических пользователей,
// заведомо не пересекающийся с настоящими (Telegram ID столько не бывает).
const loadtestUserBase = 900_000_000_000

func main() {
	scenario := flag.String("scenario", "banprofile", "сценарий: banprofile | cacheban | buyload")
	users := flag.Int("users", 500, "число засеянных синтетических пользователей")
	concurrency := flag.Int("concurrency", 15, "число одновременных воркеров (см. maxConcurrentUpdates)")
	duration := flag.Duration("duration", 20*time.Second, "длительность прогона")
	stock := flag.Int("stock", 2000, "buyload: сток товара, шт.")
	maxQty := flag.Int("max-qty", 5, "buyload: максимальное количество за один Buy()")
	pgHost := flag.String("pg-host", "localhost", "")
	pgPort := flag.Int("pg-port", 5432, "")
	pgUser := flag.String("pg-user", "shop_user", "")
	pgPassword := flag.String("pg-password", "shop_pass", "")
	pgName := flag.String("pg-name", "shop_db", "")
	redisAddr := flag.String("redis-addr", "localhost:6379", "")
	redisUsername := flag.String("redis-username", "", "пусто = аутентификация только по паролю")
	redisPassword := flag.String("redis-password", "", "")
	flag.Parse()

	log := zap.NewNop().Sugar() // сама нагрузка не должна тонуть в логах сервисов

	ctx := context.Background()

	db, err := pgdb.NewClient(&config.PostgresConfig{
		DBHost: *pgHost, DBPort: *pgPort, DBUser: *pgUser, DBPassword: *pgPassword, DBName: *pgName, DBSSLMode: "disable",
	}, log)
	if err != nil {
		fatalf("postgres: %v", err)
	}
	if err = pgdb.AutoMigrate(ctx, db, log, 1); err != nil {
		fatalf("automigrate: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: *redisAddr, Username: *redisUsername, Password: *redisPassword})
	if err = redisClient.Ping(ctx).Err(); err != nil {
		fatalf("redis: %v", err)
	}
	cache := rdb.NewRedisCache(redisClient, log)

	userRepo := pgdb.NewUserRepo(db, log)
	userService := svc.NewUserSrv(userRepo, cache, log)

	switch *scenario {
	case "banprofile":
		fmt.Printf("seeding %d synthetic users...\n", *users)
		seedUsers(ctx, userRepo, *users, 0)
		runBanProfile(ctx, userService, *users, *concurrency, *duration)
	case "cacheban":
		fmt.Printf("seeding %d synthetic users...\n", *users)
		seedUsers(ctx, userRepo, *users, 0)
		runCacheBanProfile(ctx, userService, *users, *concurrency, *duration)
	case "buyload":
		fmt.Printf("seeding %d synthetic users with balance...\n", *users)
		seedUsers(ctx, userRepo, *users, 1_000_000)

		productRepo := pgdb.NewProductRepo(db, log)
		purchaseRepo := pgdb.NewPurchaseRepo(db, log)
		categoryRepo := pgdb.NewCategoryRepo(db, log)
		replenishmentRepo := pgdb.NewReplenishmentRepo(db, log)
		settingsRepo := pgdb.NewSettingsRepo(db, log)
		transactor := pgdb.NewGormTransactor(db, log)
		settingsService := svc.NewSettingsSrv(settingsRepo, cache, log)
		purchaseService := svc.NewPurchaseSrv(userRepo, productRepo, purchaseRepo, categoryRepo, replenishmentRepo, transactor, settingsService, cache, log)

		fmt.Printf("seeding product with %d stock items...\n", *stock)
		productID := seedProduct(ctx, productRepo, *stock)

		runBuyLoad(ctx, db, purchaseService, productID, *stock, *users, *concurrency, *maxQty, *duration)
	default:
		fatalf("unknown scenario %q", *scenario)
	}
}

// seedUsers создаёт n синтетических пользователей; topUp>0 — добавляет эту
// сумму на баланс каждого (для buyload, где реально списывается).
// UpdateBalance — это "+= delta", а не "=", так что на уже существующем с
// прошлого прогона юзере баланс просто продолжит расти — для одноразового
// локального прогона это не проблема, а не баг: он не должен стать
// ограничителем сам по себе.
func seedUsers(ctx context.Context, userRepo *pgdb.UserRepo, n int, topUp float64) {
	for i := range n {
		id := int64(loadtestUserBase + i)
		user := &models.User{TelegramID: id, Username: fmt.Sprintf("loadtest_%d", i), Language: "ru"}
		// Ошибка тут обычно значит "уже существует с прошлого прогона" — не
		// повод останавливаться, баланс всё равно доливаем ниже.
		_ = userRepo.Create(ctx, user)
		if topUp > 0 {
			_ = userRepo.UpdateBalance(ctx, id, topUp)
		}
	}
}

// seedProduct создаёт новый товар с stock непроданными единицами — каждый
// прогон buyload заводит свой, старые от прошлых прогонов не мешают
// (у каждого свой product_id, посчитать проданное можно только по нему).
func seedProduct(ctx context.Context, productRepo *pgdb.ProductRepo, stock int) int64 {
	product := &models.Product{Name: fmt.Sprintf("loadtest-product-%d", time.Now().UnixNano()), Price: 10.00, IsActive: true}
	if err := productRepo.Create(ctx, product); err != nil {
		fatalf("seed product: %v", err)
	}

	// Чанками — AddItems кладёт весь срез в один INSERT (CreateInBatches с
	// batchSize=len), а Postgres ограничивает extended-протокол 65535
	// параметрами на запрос; при stock в сотни тысяч это иначе бьётся о лимит
	// на первом же вызове.
	const seedChunk = 5000
	for start := 0; start < stock; start += seedChunk {
		end := min(start+seedChunk, stock)
		contents := make([]string, end-start)
		for i := range contents {
			contents[i] = fmt.Sprintf("item-%d", start+i)
		}
		if err := productRepo.AddItems(ctx, product.ID, contents); err != nil {
			fatalf("seed stock: %v", err)
		}
	}
	return product.ID
}

// runBuyLoad — Tier 2: concurrency воркеров одновременно покупают один и тот
// же товар случайными партиями (1..maxQty), пока не кончится duration или
// сток. Проверяет не только скорость, но и то, что ReserveItems (батчем,
// FOR UPDATE SKIP LOCKED) не даёт переспродать сток при реальной конкуренции —
// это и есть тот инвариант, ради которого вообще нужен настоящий Postgres,
// а не фейковый репозиторий в unit-тесте.
func runBuyLoad(ctx context.Context, db *gorm.DB, purchaseService *svc.PurchaseSrv, productID int64, stock, userCount, concurrency, maxQty int, duration time.Duration) {
	fmt.Printf("scenario=buyload concurrency=%d duration=%s stock=%d max_qty=%d\n\n", concurrency, duration, stock, maxQty)

	var (
		wg         sync.WaitGroup
		soldByUs   atomic.Int64 // сумма quantity успешных Buy() — наш собственный счёт
		outOfStock atomic.Int64
		otherErr   atomic.Int64
		durs       []time.Duration
		mu         sync.Mutex
	)

	stopAt := time.Now().Add(duration)
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(seed))
			for time.Now().Before(stopAt) {
				userID := int64(loadtestUserBase + rnd.Intn(userCount))
				qty := 1 + rnd.Intn(maxQty)

				start := time.Now()
				purchases, _, err := purchaseService.Buy(ctx, userID, productID, qty)
				elapsed := time.Since(start)

				mu.Lock()
				durs = append(durs, elapsed)
				mu.Unlock()

				switch {
				case err == nil:
					soldByUs.Add(int64(len(purchases)))
				case errors.Is(err, domainerrors.ErrProductOutOfStock):
					outOfStock.Add(1)
				default:
					otherErr.Add(1)
				}
			}
		}(int64(w))
	}
	wg.Wait()

	// Источник истины — сама база: сколько единиц этого товара реально
	// помечено проданными, независимо от того, что насчитал сам харнесс.
	var actuallySold int64
	if err := db.WithContext(ctx).Table("product_items").
		Where("product_id = ? AND is_sold = ?", productID, true).
		Count(&actuallySold).Error; err != nil {
		fatalf("verify sold count: %v", err)
	}

	elapsedSec := duration.Seconds()
	fmt.Printf("buys ok: sold=%d (our count) out_of_stock=%d other_errors=%d\n", soldByUs.Load(), outOfStock.Load(), otherErr.Load())
	fmt.Printf("verify: product_items.is_sold=true in DB = %d (stock was %d)\n", actuallySold, stock)
	if actuallySold != soldByUs.Load() {
		fmt.Printf("MISMATCH: harness counted %d sold, DB shows %d — overselling or lost purchase!\n", soldByUs.Load(), actuallySold)
	} else if actuallySold > int64(stock) {
		fmt.Printf("OVERSOLD: %d sold against a stock of %d!\n", actuallySold, stock)
	} else {
		fmt.Printf("OK: no overselling, harness count matches DB exactly.\n")
	}
	printStats("Buy", durs)
	fmt.Printf("\n%.0f buy calls/sec, %.0f items/sec\n", float64(len(durs))/elapsedSec, float64(soldByUs.Load())/elapsedSec)
}

// opResult — одна замеренная операция: что именно (IsBanned/GetProfile) и
// сколько заняла. Раздельно, а не суммарно — чтобы увидеть вклад каждой в
// итоге, а не только общую latency "апдейта".
type opResult struct {
	op  string
	dur time.Duration
	err bool
}

// runScenario гоняет iterFn (одна "обработка апдейта" — сколько бы шагов она
// ни делала, каждый шаг сам репортит себя в resultsCh под своим op) в
// concurrency воркерах до истечения duration, потом печатает статистику по
// каждому op из opLabels отдельно — так виден вклад каждого похода в общую
// цену апдейта, не только сумма.
func runScenario(name string, opLabels []string, concurrency int, duration time.Duration,
	iterFn func(resultsCh chan<- opResult, rnd *rand.Rand)) {
	fmt.Printf("scenario=%s concurrency=%d duration=%s\n\n", name, concurrency, duration)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make(map[string][]time.Duration, len(opLabels))
		errCnt  atomic.Int64
		total   atomic.Int64
	)

	resultsCh := make(chan opResult, 4096)
	done := make(chan struct{})
	go func() {
		for r := range resultsCh {
			mu.Lock()
			results[r.op] = append(results[r.op], r.dur)
			mu.Unlock()
			if r.err {
				errCnt.Add(1)
			}
			total.Add(1)
		}
		close(done)
	}()

	stopAt := time.Now().Add(duration)
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(seed))
			for time.Now().Before(stopAt) {
				iterFn(resultsCh, rnd)
			}
		}(int64(w))
	}

	wg.Wait()
	close(resultsCh)
	<-done

	elapsed := duration.Seconds()
	fmt.Printf("total ops: %d (%.0f ops/sec), errors: %d\n\n", total.Load(), float64(total.Load())/elapsed, errCnt.Load())

	for _, op := range opLabels {
		printStats(op, results[op])
	}

	updates := total.Load() / int64(len(opLabels)) // одна "обработка апдейта" = len(opLabels) шагов
	fmt.Printf("\nsimulated updates: %d (%.0f updates/sec)\n", updates, float64(updates)/elapsed)
}

// runBanProfile — воспроизводит ТЕКУЩИЙ bot/middleware.BanCheck: один свежий
// поход (GetFreshProfile), результат кладём в ctx тем же способом, что и
// BanCheck (domainservice.WithUser) — GetProfile ниже должен обойтись без
// похода в кэш/Postgres вовсе.
func runBanProfile(ctx context.Context, userService *svc.UserSrv, userCount, concurrency int, duration time.Duration) {
	runScenario("banprofile", []string{"GetFreshProfile", "GetProfile"}, concurrency, duration,
		func(resultsCh chan<- opResult, rnd *rand.Rand) {
			id := int64(loadtestUserBase + rnd.Intn(userCount))

			start := time.Now()
			user, err := userService.GetFreshProfile(ctx, id)
			resultsCh <- opResult{op: "GetFreshProfile", dur: time.Since(start), err: err != nil}
			if err != nil {
				return
			}

			reqCtx := domainservice.WithUser(ctx, user)
			start = time.Now()
			_, err = userService.GetProfile(reqCtx, id)
			resultsCh <- opResult{op: "GetProfile", dur: time.Since(start), err: err != nil}
		})
}

// runCacheBanProfile — альтернатива: BanCheck читает через кэш (GetProfile)
// вместо гарантированно свежего GetFreshProfile, полагаясь на то, что
// BanUser инвалидирует кэш синхронно. Без ctx — обоим шагам ничего не
// передаётся, второй GetProfile просто попадает в уже тёплый после первого
// шага кэш. Компромисс: бан бьёт мгновенно только пока инвалидация не
// подвела; при сбое (best-effort, см. internal/service/cache.go:logInvalidation)
// — с задержкой до userTTL.
func runCacheBanProfile(ctx context.Context, userService *svc.UserSrv, userCount, concurrency int, duration time.Duration) {
	runScenario("cacheban", []string{"GetProfile(1st)", "GetProfile(2nd)"}, concurrency, duration,
		func(resultsCh chan<- opResult, rnd *rand.Rand) {
			id := int64(loadtestUserBase + rnd.Intn(userCount))

			start := time.Now()
			_, err := userService.GetProfile(ctx, id)
			resultsCh <- opResult{op: "GetProfile(1st)", dur: time.Since(start), err: err != nil}
			if err != nil {
				return
			}

			start = time.Now()
			_, err = userService.GetProfile(ctx, id)
			resultsCh <- opResult{op: "GetProfile(2nd)", dur: time.Since(start), err: err != nil}
		})
}

func printStats(label string, durs []time.Duration) {
	if len(durs) == 0 {
		fmt.Printf("%-12s no samples\n", label)
		return
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	p := func(pct float64) time.Duration { return durs[int(float64(len(durs)-1)*pct)] }
	var sum time.Duration
	for _, d := range durs {
		sum += d
	}
	mean := sum / time.Duration(len(durs))
	fmt.Printf("%-12s n=%-8d mean=%-10s p50=%-10s p95=%-10s p99=%-10s max=%s\n",
		label, len(durs), mean, p(0.50), p(0.95), p(0.99), durs[len(durs)-1])
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
