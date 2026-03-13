package context

import (
	"context"
	"testing"
	"time"

	"goagent/internal/core/models"
)

func TestSessionMemory(t *testing.T) {
	t.Run("create session memory", func(t *testing.T) {
		memory := NewSessionMemory(100, time.Minute)

		if memory == nil {
			t.Errorf("memory should not be nil")
		}
	})

	t.Run("set and get session", func(t *testing.T) {
		memory := NewSessionMemory(100, time.Minute)
		messages := []Message{{Role: "user", Content: "hello"}}

		err := memory.Set(context.Background(), "sess1", "user1", messages)
		if err != nil {
			t.Errorf("set error: %v", err)
		}

		data, exists := memory.Get(context.Background(), "sess1")
		if !exists {
			t.Errorf("session should exist")
		}
		if data.UserID != "user1" {
			t.Errorf("expected user1, got %s", data.UserID)
		}
	})

	t.Run("add message", func(t *testing.T) {
		memory := NewSessionMemory(100, time.Minute)
		memory.Set(context.Background(), "sess1", "user1", nil)

		err := memory.AddMessage(context.Background(), "sess1", Message{Role: "user", Content: "test"})
		if err != nil {
			t.Errorf("add message error: %v", err)
		}
	})

	t.Run("delete session", func(t *testing.T) {
		memory := NewSessionMemory(100, time.Minute)
		memory.Set(context.Background(), "sess1", "user1", nil)

		err := memory.Delete(context.Background(), "sess1")
		if err != nil {
			t.Errorf("delete error: %v", err)
		}

		_, exists := memory.Get(context.Background(), "sess1")
		if exists {
			t.Errorf("session should not exist after delete")
		}
	})

	t.Run("size", func(t *testing.T) {
		memory := NewSessionMemory(100, time.Minute)
		memory.Set(context.Background(), "sess1", "user1", nil)

		if memory.Size() != 1 {
			t.Errorf("expected size 1, got %d", memory.Size())
		}
	})
}

func TestUserMemory(t *testing.T) {
	t.Run("create user memory", func(t *testing.T) {
		memory := NewUserMemory(100)

		if memory == nil {
			t.Errorf("memory should not be nil")
		}
	})

	t.Run("set and get user", func(t *testing.T) {
		memory := NewUserMemory(100)
		profile := &models.UserProfile{UserID: "user1"}

		err := memory.Set(context.Background(), "user1", profile)
		if err != nil {
			t.Errorf("set error: %v", err)
		}

		data, exists := memory.Get(context.Background(), "user1")
		if !exists {
			t.Errorf("user should exist")
		}
		_ = data
	})
}

func TestCache(t *testing.T) {
	t.Run("create cache", func(t *testing.T) {
		cache := NewCache(100, time.Minute)

		if cache == nil {
			t.Errorf("cache should not be nil")
		}
	})

	t.Run("set and get", func(t *testing.T) {
		cache := NewCache(100, time.Minute)

		cache.Set(context.Background(), "key1", "value1")

		val, exists := cache.Get(context.Background(), "key1")
		if !exists {
			t.Errorf("key should exist")
		}
		if val != "value1" {
			t.Errorf("expected value1, got %v", val)
		}
	})

	t.Run("delete", func(t *testing.T) {
		cache := NewCache(100, time.Minute)
		cache.Set(context.Background(), "key1", "value1")

		cache.Delete(context.Background(), "key1")

		_, exists := cache.Get(context.Background(), "key1")
		if exists {
			t.Errorf("key should not exist after delete")
		}
	})
}

func TestLRUCache(t *testing.T) {
	t.Run("create LRU cache", func(t *testing.T) {
		cache := NewLRUCache(2)

		if cache == nil {
			t.Errorf("cache should not be nil")
		}
	})

	t.Run("set and get", func(t *testing.T) {
		cache := NewLRUCache(2)
		cache.Set(context.Background(), "key1", "value1")

		val, exists := cache.Get(context.Background(), "key1")
		if !exists {
			t.Errorf("key should exist")
		}
		if val != "value1" {
			t.Errorf("expected value1, got %v", val)
		}
	})

	t.Run("size after eviction", func(t *testing.T) {
		cache := NewLRUCache(2)
		cache.Set(context.Background(), "key1", "value1")
		cache.Set(context.Background(), "key2", "value2")
		cache.Set(context.Background(), "key3", "value3")

		// Size should be 2 after eviction
		if cache.Size() != 2 {
			t.Errorf("expected size 2, got %d", cache.Size())
		}
	})
}
