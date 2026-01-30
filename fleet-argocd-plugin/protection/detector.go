// Copyright 2024 Abridge AI Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Transient issue detection for GKE Fleet API responses
// Prevents application deletions from incomplete API responses

package protection

import (
	"fmt"
	"sync"
	"time"
)

// Snapshot represents a point-in-time Fleet API response
type Snapshot struct {
	ItemCount int
	Timestamp time.Time
}

// Detector identifies transient Fleet API issues via pattern analysis
type Detector struct {
	history []Snapshot
	mu      sync.Mutex

	// Configuration
	windowDuration       time.Duration
	oscillationThreshold int
	dropThreshold        float64
}

// NewDetector creates a new transient issue detector
// windowDuration: how far back to look for patterns (e.g., 10 minutes)
// oscillationThreshold: number of count changes indicating instability (e.g., 2)
// dropThreshold: percent decrease indicating sudden drop (e.g., 0.3 for 30%)
func NewDetector(windowDuration time.Duration, oscillationThreshold int, dropThreshold float64) *Detector {
	return &Detector{
		history:              []Snapshot{},
		windowDuration:       windowDuration,
		oscillationThreshold: oscillationThreshold,
		dropThreshold:        dropThreshold,
	}
}

// IsTransientIssue checks if current response indicates a transient API issue
// Returns: (isTransient bool, reason string)
func (d *Detector) IsTransientIssue(currentCount int) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add current snapshot
	snapshot := Snapshot{
		ItemCount: currentCount,
		Timestamp: time.Now(),
	}
	d.history = append(d.history, snapshot)

	// Keep only snapshots within window
	cutoff := time.Now().Add(-d.windowDuration)
	filtered := []Snapshot{}
	for _, s := range d.history {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
		}
	}
	d.history = filtered

	// Need at least 3 data points to detect patterns
	if len(d.history) < 3 {
		return false, ""
	}

	// Check for oscillation (count changing multiple times)
	changes := 0
	for i := 1; i < len(d.history); i++ {
		if d.history[i].ItemCount != d.history[i-1].ItemCount {
			changes++
		}
	}

	if changes >= d.oscillationThreshold {
		pattern := d.formatPattern()
		reason := fmt.Sprintf(
			"Oscillation detected: %d count changes in last %.0f minutes. Pattern: %s",
			changes,
			d.windowDuration.Minutes(),
			pattern,
		)
		return true, reason
	}

	// Check for sudden significant drop
	if len(d.history) >= 2 {
		// Calculate average of previous snapshots
		sum := 0
		count := 0
		for i := 0; i < len(d.history)-1; i++ {
			sum += d.history[i].ItemCount
			count++
		}

		if count > 0 {
			avgPrev := float64(sum) / float64(count)

			if avgPrev > 0 {
				decrease := (avgPrev - float64(currentCount)) / avgPrev
				if decrease > d.dropThreshold {
					reason := fmt.Sprintf(
						"Sudden drop detected: %.0f%% decrease (avg %.0f to %d)",
						decrease*100,
						avgPrev,
						currentCount,
					)
					return true, reason
				}
			}
		}
	}

	return false, ""
}

// formatPattern returns a human-readable pattern string
func (d *Detector) formatPattern() string {
	pattern := ""
	for i, s := range d.history {
		if i > 0 {
			pattern += " → "
		}
		pattern += fmt.Sprintf("%d@%s", s.ItemCount, s.Timestamp.Format("15:04:05"))
	}
	return pattern
}

// GetHistory returns the current history for debugging
func (d *Detector) GetHistory() []Snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Snapshot{}, d.history...)
}
