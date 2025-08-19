package domain

import (
	"testing"
	"time"
)

func TestNewCacheEntry(t *testing.T) {
	ttl := 5 * time.Minute
	entry := NewCacheEntry(ttl)

	if entry.TTL() != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, entry.TTL())
	}

	// Check that fetchedAt is recent (within 1 second)
	now := time.Now()
	if now.Sub(entry.FetchedAt()) > time.Second {
		t.Errorf("FetchedAt should be recent, got %v ago", now.Sub(entry.FetchedAt()))
	}
}

func TestNewCacheEntryAt(t *testing.T) {
	fetchedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute
	entry := NewCacheEntryAt(fetchedAt, ttl)

	if entry.FetchedAt() != fetchedAt {
		t.Errorf("Expected fetchedAt %v, got %v", fetchedAt, entry.FetchedAt())
	}

	if entry.TTL() != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, entry.TTL())
	}
}

func TestCacheEntry_IsFresh(t *testing.T) {
	tests := []struct {
		name      string
		fetchedAt time.Time
		ttl       time.Duration
		checkAt   time.Time
		want      bool
	}{
		{
			name:      "fresh entry",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 5, 0, 0, time.UTC), // 5 minutes later
			want:      true,
		},
		{
			name:      "expired entry",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 15, 0, 0, time.UTC), // 15 minutes later
			want:      false,
		},
		{
			name:      "exactly at TTL",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 10, 0, 0, time.UTC), // exactly 10 minutes later
			want:      false,
		},
		{
			name:      "zero TTL",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       0,
			checkAt:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), // same time
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewCacheEntryAt(tt.fetchedAt, tt.ttl)
			got := entry.IsFreshAt(tt.checkAt)
			if got != tt.want {
				t.Errorf("IsFreshAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheEntry_IsExpired(t *testing.T) {
	fetchedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute
	entry := NewCacheEntryAt(fetchedAt, ttl)

	// Test fresh entry
	checkAt := time.Date(2023, 1, 1, 12, 5, 0, 0, time.UTC) // 5 minutes later
	if entry.IsExpiredAt(checkAt) {
		t.Error("Entry should not be expired after 5 minutes with 10 minute TTL")
	}

	// Test expired entry
	checkAt = time.Date(2023, 1, 1, 12, 15, 0, 0, time.UTC) // 15 minutes later
	if !entry.IsExpiredAt(checkAt) {
		t.Error("Entry should be expired after 15 minutes with 10 minute TTL")
	}
}

func TestCacheEntry_ExpiresAt(t *testing.T) {
	fetchedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute
	entry := NewCacheEntryAt(fetchedAt, ttl)

	expected := time.Date(2023, 1, 1, 12, 10, 0, 0, time.UTC)
	if entry.ExpiresAt() != expected {
		t.Errorf("Expected ExpiresAt %v, got %v", expected, entry.ExpiresAt())
	}
}

func TestCacheEntry_Age(t *testing.T) {
	fetchedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute
	entry := NewCacheEntryAt(fetchedAt, ttl)

	checkAt := time.Date(2023, 1, 1, 12, 7, 0, 0, time.UTC) // 7 minutes later
	expectedAge := 7 * time.Minute

	if entry.AgeAt(checkAt) != expectedAge {
		t.Errorf("Expected age %v, got %v", expectedAge, entry.AgeAt(checkAt))
	}
}

func TestCacheEntry_RemainingTTL(t *testing.T) {
	tests := []struct {
		name      string
		fetchedAt time.Time
		ttl       time.Duration
		checkAt   time.Time
		want      time.Duration
	}{
		{
			name:      "fresh entry with remaining time",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 3, 0, 0, time.UTC), // 3 minutes later
			want:      7 * time.Minute,                              // 10 - 3 = 7 minutes remaining
		},
		{
			name:      "expired entry returns zero",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 15, 0, 0, time.UTC), // 15 minutes later
			want:      0,
		},
		{
			name:      "exactly at expiration returns zero",
			fetchedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ttl:       10 * time.Minute,
			checkAt:   time.Date(2023, 1, 1, 12, 10, 0, 0, time.UTC), // exactly 10 minutes later
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewCacheEntryAt(tt.fetchedAt, tt.ttl)
			got := entry.RemainingTTLAt(tt.checkAt)
			if got != tt.want {
				t.Errorf("RemainingTTLAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheEntry_Refresh(t *testing.T) {
	originalFetchedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute
	entry := NewCacheEntryAt(originalFetchedAt, ttl)

	// Refresh to a new time
	newFetchedAt := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
	entry.RefreshAt(newFetchedAt)

	if entry.FetchedAt() != newFetchedAt {
		t.Errorf("Expected fetchedAt to be refreshed to %v, got %v", newFetchedAt, entry.FetchedAt())
	}

	// TTL should remain the same
	if entry.TTL() != ttl {
		t.Errorf("Expected TTL to remain %v, got %v", ttl, entry.TTL())
	}
}

func TestCacheEntry_EdgeCases(t *testing.T) {
	t.Run("negative TTL", func(t *testing.T) {
		entry := NewCacheEntry(-5 * time.Minute)
		// Negative TTL should make entry immediately expired
		if entry.IsFresh() {
			t.Error("Entry with negative TTL should not be fresh")
		}
	})

	t.Run("zero TTL", func(t *testing.T) {
		entry := NewCacheEntry(0)
		// Zero TTL should make entry immediately expired
		if entry.IsFresh() {
			t.Error("Entry with zero TTL should not be fresh")
		}
	})
}
