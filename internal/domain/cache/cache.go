// Package cache — read-through кэш перед репозиториями, по интерфейсу на
// сущность (user_cache.go, product_cache.go, category_cache.go), без общего типа.
package cache

import "errors"

// ErrMiss — ключ отсутствует или истёк, вызывающий идёт за данными в репозиторий.
var ErrMiss = errors.New("cache: key not found")
