package provider

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCacheFetchesEachKeyOnceWithinTTL(t *testing.T) {
	cache := NewCache[string](0)
	calls := 0
	fetch := func(context.Context) (string, error) { calls++; return "value", nil }
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil || calls != 1 {
		t.Fatalf("cache calls=%d err=%v", calls, err)
	}
}

func TestCacheBoundsRepositoryKeyChurn(t *testing.T) {
	cache := NewCache[int](time.Hour)
	for index := 0; index < maxProviderCacheEntries+44; index++ {
		key := fmt.Sprintf("repository-%03d", index)
		if _, err := cache.Get(context.Background(), key, func(context.Context) (int, error) { return index, nil }); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(cache.items); got != maxProviderCacheEntries {
		t.Fatalf("cache size = %d, want %d", got, maxProviderCacheEntries)
	}
}
