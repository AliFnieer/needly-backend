package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		t.Skip("REDIS_HOST not set, skipping cache integration tests")
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	return client
}

func TestCache_SetGetRoundTrip(t *testing.T) {
	client := redisClient(t)
	c := NewCache(client, 30*time.Second)
	defer client.Close()

	type item struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	key := fmt.Sprintf("test:setget:%d", time.Now().UnixNano())
	want := item{Name: "hello", Value: 42}

	if err := c.Set(context.Background(), key, &want); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var got item
	ok, err := c.Get(context.Background(), key, &got)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Name != want.Name || got.Value != want.Value {
		t.Fatalf("expected %+v, got %+v", want, got)
	}

	// Cleanup
	_ = c.Delete(context.Background(), key)
}

func TestCache_CacheMiss(t *testing.T) {
	client := redisClient(t)
	c := NewCache(client, 30*time.Second)
	defer client.Close()

	key := fmt.Sprintf("test:miss:%d", time.Now().UnixNano())
	var dest string
	ok, err := c.Get(context.Background(), key, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss (false)")
	}
}

func TestCache_Delete(t *testing.T) {
	client := redisClient(t)
	c := NewCache(client, 30*time.Second)
	defer client.Close()

	key := fmt.Sprintf("test:delete:%d", time.Now().UnixNano())
	if err := c.Set(context.Background(), key, "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	var dest string
	ok, err := c.Get(context.Background(), key, &dest)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit before delete")
	}

	// Delete
	if err := c.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	ok, err = c.Get(context.Background(), key, &dest)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss after delete")
	}
}
