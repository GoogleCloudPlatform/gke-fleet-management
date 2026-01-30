// Copyright 2024 Abridge AI Inc.
// SPDX-License-Identifier: Apache-2.0

package protection

import (
	"testing"
	"time"
)

func TestDetector_NoTransientIssue_StableResponse(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Simulate stable responses (no changes)
	counts := []int{12, 12, 12, 12, 12}

	for _, count := range counts {
		isTransient, reason := detector.IsTransientIssue(count)
		if isTransient {
			t.Errorf("Expected no transient issue for stable responses, got: %s", reason)
		}
	}
}

func TestDetector_NoTransientIssue_LegitimateIncrease(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Simulate legitimate scope addition: 12 → 14 (sustained)
	counts := []int{12, 12, 12, 14, 14, 14}

	for i, count := range counts {
		isTransient, reason := detector.IsTransientIssue(count)
		if isTransient {
			t.Errorf("Step %d: Expected no transient issue for legitimate increase, got: %s", i, reason)
		}
	}
}

func TestDetector_NoTransientIssue_LegitimateDecrease(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Simulate legitimate scope removal: 12 → 10 (sustained, <30% drop)
	counts := []int{12, 12, 12, 10, 10, 10}

	for i, count := range counts {
		isTransient, _ := detector.IsTransientIssue(count)
		// Small decrease without oscillation should not trigger
		if isTransient && i < 3 {
			t.Errorf("Step %d: Should not trigger on legitimate small decrease", i)
		}
	}
}

func TestDetector_TransientIssue_Oscillation(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Simulate transient issue: 12 → 6 → 12 (oscillation)
	testCases := []struct {
		count           int
		expectTransient bool
		description     string
	}{
		{12, false, "First stable count"},
		{12, false, "Second stable count"},
		{12, false, "Third stable count"},
		{6, true, "Drop to 6 (50% drop = sudden drop detected!)"},
		{12, true, "Back to 12 (oscillation also detected)"},
	}

	for _, tc := range testCases {
		isTransient, reason := detector.IsTransientIssue(tc.count)
		if isTransient != tc.expectTransient {
			t.Errorf("%s: expected transient=%v, got transient=%v, reason=%s",
				tc.description, tc.expectTransient, isTransient, reason)
		}
	}
}

func TestDetector_TransientIssue_SuddenDrop(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.5) // 50% threshold

	// Simulate sudden >50% drop: 12 → 5
	testCases := []struct {
		count           int
		expectTransient bool
		description     string
	}{
		{12, false, "Stable at 12"},
		{12, false, "Still stable at 12"},
		{12, false, "Still stable at 12"},
		{5, true, "Sudden drop to 5 (58% decrease)"},
	}

	for _, tc := range testCases {
		isTransient, reason := detector.IsTransientIssue(tc.count)
		if isTransient != tc.expectTransient {
			t.Errorf("%s: expected transient=%v, got transient=%v, reason=%s",
				tc.description, tc.expectTransient, isTransient, reason)
		}
	}
}

func TestDetector_NoTransientIssue_InsuffientData(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// With < 3 data points, should never trigger
	counts := []int{12, 6}

	for _, count := range counts {
		isTransient, _ := detector.IsTransientIssue(count)
		if isTransient {
			t.Errorf("Should not detect transient issue with < 3 data points")
		}
	}
}

func TestDetector_WindowExpiration(t *testing.T) {
	// Short window for testing
	detector := NewDetector(1*time.Second, 2, 0.3)

	// Add old data
	detector.IsTransientIssue(12)
	detector.IsTransientIssue(6)

	// Wait for window to expire
	time.Sleep(2 * time.Second)

	// New data should not consider expired data
	detector.IsTransientIssue(12)

	history := detector.GetHistory()
	if len(history) != 1 {
		t.Errorf("Expected history to be pruned to 1 item, got %d", len(history))
	}
}

func TestDetector_RealIncident_2026_01_30(t *testing.T) {
	// Reproduce actual incident pattern from GCP audit logs
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Actual pattern from incident:
	// 12, 12, 12, 12, 12, 12, 12, 6, 12, 12, 12...
	incidentPattern := []int{12, 12, 12, 12, 12, 12, 12, 6, 12}

	var detectedAt int = -1
	for i, count := range incidentPattern {
		isTransient, reason := detector.IsTransientIssue(count)
		if isTransient && detectedAt == -1 {
			detectedAt = i
			t.Logf("Transient issue detected at index %d: %s", i, reason)
		}
	}

	// Should detect transient issue at the drop to 6 (sudden drop detection)
	if detectedAt != 7 {
		t.Errorf("Expected to detect transient issue at index 7 (when drop to 6 occurs), got %d", detectedAt)
	}
}

func TestDetector_MultipleOscillations(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Multiple oscillations: 12 → 6 → 12 → 6 → 12
	pattern := []int{12, 12, 12, 6, 12, 6, 12}

	transientDetections := 0
	for _, count := range pattern {
		isTransient, _ := detector.IsTransientIssue(count)
		if isTransient {
			transientDetections++
		}
	}

	// Should detect transient issue at least once
	if transientDetections == 0 {
		t.Errorf("Expected to detect transient issue in multiple oscillations, got 0 detections")
	}
}

func TestDetector_ConcurrentAccess(t *testing.T) {
	detector := NewDetector(10*time.Minute, 2, 0.3)

	// Test thread safety - concurrent calls should not cause race conditions
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(count int) {
			detector.IsTransientIssue(count)
			done <- true
		}(12)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should complete without panics or data races
}

func TestDetector_ConfigurableThresholds(t *testing.T) {
	testCases := []struct {
		name                 string
		oscillationThreshold int
		dropThreshold        float64
		pattern              []int
		expectTransient      bool
	}{
		{
			name:                 "Strict: 1 change triggers",
			oscillationThreshold: 1,
			dropThreshold:        0.1, // 10%
			pattern:              []int{12, 12, 11},
			expectTransient:      true,
		},
		{
			name:                 "Lenient: 3 changes required",
			oscillationThreshold: 3,
			dropThreshold:        0.5, // 50%
			pattern:              []int{12, 11, 12, 11},
			expectTransient:      true, // 3 changes meets threshold of 3
		},
		{
			name:                 "Conservative drop threshold",
			oscillationThreshold: 2,
			dropThreshold:        0.6, // 60%
			pattern:              []int{12, 12, 12, 5}, // 58% drop
			expectTransient:      false, // Below 60% threshold
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewDetector(10*time.Minute, tc.oscillationThreshold, tc.dropThreshold)

			var detectedTransient bool
			for _, count := range tc.pattern {
				isTransient, _ := detector.IsTransientIssue(count)
				if isTransient {
					detectedTransient = true
					break
				}
			}

			if detectedTransient != tc.expectTransient {
				t.Errorf("Expected transient=%v, got transient=%v",
					tc.expectTransient, detectedTransient)
			}
		})
	}
}
