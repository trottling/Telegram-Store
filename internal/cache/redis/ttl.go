package redis

import "time"

const (
	userTTL             = 10 * time.Minute
	activeProductsTTL   = time.Minute
	productTTL          = time.Minute
	productCountTTL     = 30 * time.Second
	categoryChildrenTTL = time.Minute
	stateTTL            = 5 * time.Minute
)
