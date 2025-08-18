// Package domain provides cache expiration logic encapsulation.
package domain

import (
	"time"
)

// CacheEntry represents a cache entry with expiration logic.
// It encapsulates cache timing concerns and provides predicate methods
// to separate policy from implementation.
type CacheEntry struct {
	fetchedAt time.Time
	ttl       time.Duration
}

// NewCacheEntry creates a new cache entry with the given TTL.
// The fetchedAt time is set to the current time.
func NewCacheEntry(ttl time.Duration) *CacheEntry {
	return &CacheEntry{
		fetchedAt: time.Now(),
		ttl:       ttl,
	}
}

// NewCacheEntryAt creates a new cache entry with the given TTL and fetch time.
// This is useful for testing or when you need to specify an exact fetch time.
func NewCacheEntryAt(fetchedAt time.Time, ttl time.Duration) *CacheEntry {
	return &CacheEntry{
		fetchedAt: fetchedAt,
		ttl:       ttl,
	}
}

// IsFresh returns true if the entry is not expired based on the current time.
// This encapsulates the common `time.Since(fetchedAt) < ttl` pattern.
func (ce *CacheEntry) IsFresh() bool {
	return ce.IsFreshAt(time.Now())
}

// IsFreshAt returns true if the entry is not expired at the given time.
// This method is useful for testing with a specific time.
func (ce *CacheEntry) IsFreshAt(now time.Time) bool {
	return now.Sub(ce.fetchedAt) < ce.ttl
}

// IsExpired returns true if the entry has expired based on the current time.
// This is the inverse of IsFresh for more readable code in some contexts.
func (ce *CacheEntry) IsExpired() bool {
	return !ce.IsFresh()
}

// IsExpiredAt returns true if the entry has expired at the given time.
// This is the inverse of IsFreshAt for more readable code in some contexts.
func (ce *CacheEntry) IsExpiredAt(now time.Time) bool {
	return !ce.IsFreshAt(now)
}

// ExpiresAt returns the expiration time of this cache entry.
func (ce *CacheEntry) ExpiresAt() time.Time {
	return ce.fetchedAt.Add(ce.ttl)
}

// Age returns how long ago this entry was fetched.
func (ce *CacheEntry) Age() time.Duration {
	return time.Since(ce.fetchedAt)
}

// AgeAt returns how long ago this entry was fetched relative to the given time.
func (ce *CacheEntry) AgeAt(now time.Time) time.Duration {
	return now.Sub(ce.fetchedAt)
}

// TTL returns the time-to-live duration for this cache entry.
func (ce *CacheEntry) TTL() time.Duration {
	return ce.ttl
}

// FetchedAt returns when this cache entry was fetched.
func (ce *CacheEntry) FetchedAt() time.Time {
	return ce.fetchedAt
}

// RemainingTTL returns how much TTL time remains based on the current time.
// Returns 0 if the entry has expired.
func (ce *CacheEntry) RemainingTTL() time.Duration {
	return ce.RemainingTTLAt(time.Now())
}

// RemainingTTLAt returns how much TTL time remains at the given time.
// Returns 0 if the entry has expired.
func (ce *CacheEntry) RemainingTTLAt(now time.Time) time.Duration {
	remaining := ce.ttl - now.Sub(ce.fetchedAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Refresh updates the fetchedAt time to the current time, effectively
// resetting the cache entry's expiration timer.
func (ce *CacheEntry) Refresh() {
	ce.RefreshAt(time.Now())
}

// RefreshAt updates the fetchedAt time to the given time, effectively
// resetting the cache entry's expiration timer to that specific time.
func (ce *CacheEntry) RefreshAt(fetchedAt time.Time) {
	ce.fetchedAt = fetchedAt
}