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

	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	svc "github.com/trottling/Telegram-Store/internal/service"
)

// loadtestUserBase — диапазон TelegramID для синтетических пользователей,
// заведомо не пересекающийся с настоящими (Telegram ID столько не бывает).
const loadtestUserBase = 900_000_000_000

func main() {
	scenario := flag.String("scenario", "banprofile", "сценарий: banprofile")
	users := flag.Int("users", 500, "число засеянных синтетических пользователей")
	concurrency := flag.Int("concurrency", 15, "число одновременных воркеров (см. maxConcurrentUpdates)")
	duration := flag.Duration("duration", 20*time.Second, "длительность прогона")
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

	fmt.Printf("seeding %d synthetic users...\n", *users)
	seedUsers(ctx, userRepo, *users)

	switch *scenario {
	case "banprofile":
		runBanProfile(ctx, userService, *users, *concurrency, *duration)
	default:
		fatalf("unknown scenario %q", *scenario)
	}
}

func seedUsers(ctx context.Context, userRepo *pgdb.UserRepo, n int) {
	for i := range n {
		id := int64(loadtestUserBase + i)
		user := &models.User{TelegramID: id, Username: fmt.Sprintf("loadtest_%d", i), Language: "ru"}
		if err := userRepo.Create(ctx, user); err != nil {
			// Уже существует с прошлого прогона — не ошибка, просто продолжаем.
			continue
		}
	}
}

// opResult — одна замеренная операция: что именно (IsBanned/GetProfile) и
// сколько заняла. Раздельно, а не суммарно — чтобы увидеть вклад каждой в
// итоге, а не только общую latency "апдейта".
type opResult struct {
	op  string
	dur time.Duration
	err bool
}

func runBanProfile(ctx context.Context, userService *svc.UserSrv, userCount, concurrency int, duration time.Duration) {
	fmt.Printf("scenario=banprofile concurrency=%d duration=%s users=%d\n\n", concurrency, duration, userCount)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make(map[string][]time.Duration, 2)
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
				id := int64(loadtestUserBase + rnd.Intn(userCount))

				// Воспроизводит текущий bot/middleware.BanCheck: один свежий
				// поход (GetFreshProfile), результат кладём в ctx тем же
				// способом, что и BanCheck (domainservice.WithUser) — GetProfile
				// ниже должен обойтись без похода в кэш/Postgres вовсе.
				start := time.Now()
				user, err := userService.GetFreshProfile(ctx, id)
				resultsCh <- opResult{op: "GetFreshProfile", dur: time.Since(start), err: err != nil}
				if err != nil {
					continue
				}

				reqCtx := domainservice.WithUser(ctx, user)
				start = time.Now()
				_, err = userService.GetProfile(reqCtx, id)
				resultsCh <- opResult{op: "GetProfile", dur: time.Since(start), err: err != nil}
			}
		}(int64(w))
	}

	wg.Wait()
	close(resultsCh)
	<-done

	elapsed := duration.Seconds()
	fmt.Printf("total ops: %d (%.0f ops/sec), errors: %d\n\n", total.Load(), float64(total.Load())/elapsed, errCnt.Load())

	for _, op := range []string{"GetFreshProfile", "GetProfile"} {
		printStats(op, results[op])
	}

	updates := total.Load() / 2 // одна "обработка апдейта" = IsBanned + GetProfile
	fmt.Printf("\nsimulated updates: %d (%.0f updates/sec)\n", updates, float64(updates)/elapsed)
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
