package workspace

import (
	"context"
	"strconv"
	"time"

	"backend/internal/model"
	"backend/pkg/cache"
)

const accessCacheTTL = 5 * time.Minute
const accessCachePrefix = "ws:access:"

// CachedAccessChecker оборачивает AccessChecker и кэширует результат HasAccess.
func CachedAccessChecker(inner AccessChecker, c cache.Cache) AccessChecker {
	if c == nil {
		return inner
	}
	return &cachedAccessChecker{inner: inner, cache: c}
}

type cachedAccessChecker struct {
	inner AccessChecker
	cache cache.Cache
}

func (c *cachedAccessChecker) HasAccess(ctx context.Context, workspaceID, userID string, userRole model.UserRole) (bool, error) {
	key := accessCachePrefix + workspaceID + ":" + userID + ":" + string(userRole)
	if v, ok := c.cache.Get(ctx, key); ok {
		allowed, _ := strconv.ParseBool(v)
		return allowed, nil
	}
	allowed, err := c.inner.HasAccess(ctx, workspaceID, userID, userRole)
	if err != nil {
		return false, err
	}
	c.cache.Set(ctx, key, strconv.FormatBool(allowed), accessCacheTTL)
	return allowed, nil
}
