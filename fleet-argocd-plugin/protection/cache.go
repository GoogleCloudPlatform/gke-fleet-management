// Copyright 2024 Abridge AI Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Response caching for GKE Fleet API
// Provides fallback data when API returns incomplete responses

package protection

import (
	"fmt"
	"log"
	"sync"
	"time"

	fleet "google.golang.org/api/gkehub/v1"
)

// Cache stores the last known good Fleet API response
type Cache struct {
	membershipBindings []*fleet.MembershipBinding
	timestamp          time.Time
	mu                 sync.RWMutex
	maxAge             time.Duration
}

// NewCache creates a new cache with specified TTL
func NewCache(maxAge time.Duration) *Cache {
	return &Cache{
		maxAge: maxAge,
	}
}

// Get returns cached data if valid
// Returns: (data []*fleet.MembershipBinding, valid bool)
func (c *Cache) Get() ([]*fleet.MembershipBinding, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.membershipBindings == nil {
		return nil, false
	}

	age := time.Since(c.timestamp)
	if age > c.maxAge {
		return nil, false
	}

	log.Printf("Using cached response: %d items (age: %v)", len(c.membershipBindings), age)
	return c.membershipBindings, true
}

// Set stores data in cache
func (c *Cache) Set(data []*fleet.MembershipBinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.membershipBindings = data
	c.timestamp = time.Now()
}

// Age returns age of cached data
func (c *Cache) Age() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.membershipBindings == nil {
		return 0
	}
	return time.Since(c.timestamp)
}

// Info returns cache information for logging
func (c *Cache) Info() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.membershipBindings == nil {
		return "Cache: empty"
	}

	age := time.Since(c.timestamp)
	return fmt.Sprintf("Cache: %d items, age=%v, expires in=%v",
		len(c.membershipBindings),
		age,
		c.maxAge-age)
}
