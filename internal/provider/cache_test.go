package provider

import (
	"context"
	"testing"
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
