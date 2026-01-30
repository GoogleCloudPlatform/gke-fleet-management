// Copyright 2024 Abridge AI Inc.
// SPDX-License-Identifier: Apache-2.0

package protection

import (
	"testing"
	"time"

	fleet "google.golang.org/api/gkehub/v1"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Create test data
	testData := []*fleet.MembershipBinding{
		{Name: "projects/123/locations/us-central1/memberships/cluster1/bindings/binding1"},
		{Name: "projects/123/locations/us-east4/memberships/cluster2/bindings/binding2"},
	}

	// Set data
	cache.Set(testData)

	// Get data
	data, ok := cache.Get()
	if !ok {
		t.Fatal("Expected cache to have data, got empty")
	}

	if len(data) != len(testData) {
		t.Errorf("Expected %d items, got %d", len(testData), len(data))
	}

	if data[0].Name != testData[0].Name {
		t.Errorf("Expected name %s, got %s", testData[0].Name, data[0].Name)
	}
}

func TestCache_GetEmpty(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Get from empty cache
	data, ok := cache.Get()
	if ok {
		t.Error("Expected cache to be empty")
	}

	if data != nil {
		t.Error("Expected nil data from empty cache")
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	testData := []*fleet.MembershipBinding{
		{Name: "test-binding"},
	}

	// Set data
	cache.Set(testData)

	// Immediately get - should work
	if _, ok := cache.Get(); !ok {
		t.Error("Expected cache to have fresh data")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get expired data - should fail
	if data, ok := cache.Get(); ok {
		t.Errorf("Expected cache to be expired, but got data: %v", data)
	}
}

func TestCache_Age(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Empty cache age should be 0
	if age := cache.Age(); age != 0 {
		t.Errorf("Expected age 0 for empty cache, got %v", age)
	}

	// Set data and check age
	testData := []*fleet.MembershipBinding{{Name: "test"}}
	cache.Set(testData)

	time.Sleep(100 * time.Millisecond)

	age := cache.Age()
	if age < 50*time.Millisecond || age > 200*time.Millisecond {
		t.Errorf("Expected age around 100ms, got %v", age)
	}
}

func TestCache_Info(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Empty cache info
	info := cache.Info()
	if info != "Cache: empty" {
		t.Errorf("Expected 'Cache: empty', got %s", info)
	}

	// Set data
	testData := []*fleet.MembershipBinding{
		{Name: "binding1"},
		{Name: "binding2"},
	}
	cache.Set(testData)

	// Cache with data info
	info = cache.Info()
	if info == "Cache: empty" {
		t.Error("Expected cache info with data")
	}
}

func TestCache_OverwritePreviousData(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Set first data
	firstData := []*fleet.MembershipBinding{
		{Name: "first-binding"},
	}
	cache.Set(firstData)

	// Set second data (overwrite)
	secondData := []*fleet.MembershipBinding{
		{Name: "second-binding-1"},
		{Name: "second-binding-2"},
	}
	cache.Set(secondData)

	// Get should return second data
	data, ok := cache.Get()
	if !ok {
		t.Fatal("Expected cache to have data")
	}

	if len(data) != 2 {
		t.Errorf("Expected 2 items from second data, got %d", len(data))
	}

	if data[0].Name != "second-binding-1" {
		t.Errorf("Expected second data, got %s", data[0].Name)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	testData := []*fleet.MembershipBinding{
		{Name: "binding1"},
	}

	// Concurrent writes and reads
	done := make(chan bool, 200)

	// 100 writers
	for i := 0; i < 100; i++ {
		go func() {
			cache.Set(testData)
			done <- true
		}()
	}

	// 100 readers
	for i := 0; i < 100; i++ {
		go func() {
			cache.Get()
			cache.Age()
			cache.Info()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 200; i++ {
		<-done
	}

	// Should complete without panics or data races
}

func TestCache_LargeDataset(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Create large dataset (simulate many membership bindings)
	largeData := make([]*fleet.MembershipBinding, 1000)
	for i := 0; i < 1000; i++ {
		largeData[i] = &fleet.MembershipBinding{
			Name: "binding-" + string(rune(i)),
		}
	}

	// Set and get large dataset
	cache.Set(largeData)

	data, ok := cache.Get()
	if !ok {
		t.Fatal("Expected cache to handle large dataset")
	}

	if len(data) != 1000 {
		t.Errorf("Expected 1000 items, got %d", len(data))
	}
}

func TestCache_RefreshExtendsExpiration(t *testing.T) {
	cache := NewCache(200 * time.Millisecond)

	testData := []*fleet.MembershipBinding{{Name: "test"}}

	// Initial set
	cache.Set(testData)
	time.Sleep(100 * time.Millisecond) // Age: 100ms

	// Refresh (set again)
	cache.Set(testData)

	// Age should reset
	age := cache.Age()
	if age > 50*time.Millisecond {
		t.Errorf("Expected age to reset after refresh, got %v", age)
	}

	// Should still be valid after original TTL
	time.Sleep(150 * time.Millisecond)

	if _, ok := cache.Get(); !ok {
		t.Error("Expected cache to be valid after refresh extended expiration")
	}
}
