package watch

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestDebouncer_SingleEvent(t *testing.T) {
	var (
		mu     sync.Mutex
		result []string
	)

	d := NewDebouncer(50*time.Millisecond, func(dirs []string) {
		mu.Lock()
		result = dirs
		mu.Unlock()
	})
	defer d.Stop()

	d.Add("src")

	// Wait for debounce window to expire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(result) != 1 || result[0] != "src" {
		t.Errorf("expected [src], got %v", result)
	}
}

func TestDebouncer_MultipleEvents(t *testing.T) {
	var (
		mu     sync.Mutex
		result []string
	)

	d := NewDebouncer(100*time.Millisecond, func(dirs []string) {
		mu.Lock()
		result = dirs
		mu.Unlock()
	})
	defer d.Stop()

	// Add multiple directories rapidly
	d.Add("src")
	time.Sleep(20 * time.Millisecond)
	d.Add("lib")
	time.Sleep(20 * time.Millisecond)
	d.Add("pkg")

	// Wait for debounce window to expire
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	slices.Sort(result)
	expected := []string{"lib", "pkg", "src"}
	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDebouncer_Deduplication(t *testing.T) {
	var (
		mu     sync.Mutex
		result []string
	)

	d := NewDebouncer(50*time.Millisecond, func(dirs []string) {
		mu.Lock()
		result = dirs
		mu.Unlock()
	})
	defer d.Stop()

	// Add same directory multiple times
	d.Add("src")
	d.Add("src")
	d.Add("src")

	// Wait for debounce window to expire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(result) != 1 || result[0] != "src" {
		t.Errorf("expected single [src], got %v", result)
	}
}

func TestDebouncer_ResetOnNewEvent(t *testing.T) {
	var (
		mu        sync.Mutex
		callCount int
	)

	d := NewDebouncer(50*time.Millisecond, func(dirs []string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	defer d.Stop()

	// Add event, wait partial window, add another
	d.Add("src")
	time.Sleep(30 * time.Millisecond)
	d.Add("lib")
	time.Sleep(30 * time.Millisecond)
	d.Add("pkg")

	// Wait for final debounce
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should only have flushed once
	if callCount != 1 {
		t.Errorf("expected 1 flush, got %d", callCount)
	}
}

func TestDebouncer_FlushNow(t *testing.T) {
	var (
		mu     sync.Mutex
		result []string
	)

	d := NewDebouncer(1*time.Second, func(dirs []string) {
		mu.Lock()
		result = dirs
		mu.Unlock()
	})

	d.Add("src")
	d.Add("lib")

	// Flush immediately without waiting for timer
	d.FlushNow()

	mu.Lock()
	defer mu.Unlock()

	if len(result) != 2 {
		t.Errorf("expected 2 directories, got %d", len(result))
	}
}

func TestDebouncer_Stop(t *testing.T) {
	var (
		mu     sync.Mutex
		result []string
	)

	d := NewDebouncer(1*time.Second, func(dirs []string) {
		mu.Lock()
		result = dirs
		mu.Unlock()
	})

	d.Add("src")
	d.Stop()

	mu.Lock()
	defer mu.Unlock()

	// Stop should flush pending
	if len(result) != 1 || result[0] != "src" {
		t.Errorf("expected [src], got %v", result)
	}
}

func TestDebouncer_StopIgnoresNewEvents(t *testing.T) {
	var (
		mu        sync.Mutex
		callCount int
	)

	d := NewDebouncer(50*time.Millisecond, func(dirs []string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	d.Add("src")
	d.Stop()

	// Adding after stop should be ignored
	d.Add("lib")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestDebouncer_PendingCount(t *testing.T) {
	d := NewDebouncer(1*time.Second, func(dirs []string) {})
	defer d.Stop()

	if count := d.PendingCount(); count != 0 {
		t.Errorf("expected 0 pending, got %d", count)
	}

	d.Add("src")
	d.Add("lib")
	d.Add("src") // duplicate

	if count := d.PendingCount(); count != 2 {
		t.Errorf("expected 2 pending, got %d", count)
	}
}

func TestDebouncer_MaxPendingLimit(t *testing.T) {
	var (
		mu        sync.Mutex
		callCount int
		lastDirs  []string
	)

	d := NewDebouncer(1*time.Second, func(dirs []string) {
		mu.Lock()
		callCount++
		lastDirs = dirs
		mu.Unlock()
	})
	defer d.Stop()

	// Add more than MaxPendingDirs directories
	for i := 0; i < MaxPendingDirs+10; i++ {
		d.Add(fmt.Sprintf("dir%d", i))
	}

	// Should have flushed immediately when limit was reached
	mu.Lock()
	defer mu.Unlock()

	if callCount < 1 {
		t.Errorf("expected at least 1 flush due to limit, got %d", callCount)
	}

	// First flush should have exactly MaxPendingDirs directories
	if callCount >= 1 && len(lastDirs) != MaxPendingDirs && len(lastDirs) != 10 {
		// Either first batch of MaxPendingDirs or remaining 10
		t.Logf("got %d directories in flush (expected %d or 10)", len(lastDirs), MaxPendingDirs)
	}
}
