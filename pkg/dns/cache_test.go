package dns

import (
	"sync"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	c := NewCache()
	if c == nil {
		t.Fatal("NewCache returned nil")
	}
}

func TestCachePutAndGet(t *testing.T) {
	c := NewCache()

	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "93.184.216.34"},
	}

	c.Put("example.com", TypeA, records)

	got := c.Get("example.com", TypeA)
	if got == nil {
		t.Fatal("Get returned nil for cached entry")
	}
	if len(got) != 1 {
		t.Fatalf("Get returned %d records, want 1", len(got))
	}
	if got[0].ParsedData != "93.184.216.34" {
		t.Errorf("ParsedData = %q, want %q", got[0].ParsedData, "93.184.216.34")
	}
}

func TestCacheGetMiss(t *testing.T) {
	c := NewCache()

	got := c.Get("nonexistent.com", TypeA)
	if got != nil {
		t.Errorf("Get for missing key returned %v, want nil", got)
	}
}

func TestCacheGetWrongType(t *testing.T) {
	c := NewCache()

	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "93.184.216.34"},
	}
	c.Put("example.com", TypeA, records)

	// Query for AAAA should miss.
	got := c.Get("example.com", TypeAAAA)
	if got != nil {
		t.Errorf("Get for wrong type returned %v, want nil", got)
	}
}

func TestCachePutEmptyRecords(t *testing.T) {
	c := NewCache()

	// Putting empty records should be a no-op.
	c.Put("example.com", TypeA, nil)
	c.Put("example.com", TypeA, []ResourceRecord{})

	got := c.Get("example.com", TypeA)
	if got != nil {
		t.Errorf("Get after empty Put returned %v, want nil", got)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewCache()

	// Create a record with a very low TTL. The cache enforces a minimum of 30s,
	// so we need to manipulate the entry directly.
	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 60, ParsedData: "1.2.3.4"},
	}
	c.Put("example.com", TypeA, records)

	// It should be retrievable now.
	got := c.Get("example.com", TypeA)
	if got == nil {
		t.Fatal("Get returned nil immediately after Put")
	}

	// Manually expire the entry.
	c.mu.Lock()
	key := cacheKey("example.com", TypeA)
	c.entries[key].ExpiresAt = time.Now().Add(-1 * time.Second)
	c.mu.Unlock()

	// Now it should be expired.
	got = c.Get("example.com", TypeA)
	if got != nil {
		t.Errorf("Get returned %v for expired entry, want nil", got)
	}
}

func TestCacheMinTTL(t *testing.T) {
	c := NewCache()

	// Records with TTL < 30 should still be cached for at least 30 seconds.
	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 1, ParsedData: "1.2.3.4"},
	}
	c.Put("example.com", TypeA, records)

	// Should still be retrievable since minimum TTL is enforced.
	got := c.Get("example.com", TypeA)
	if got == nil {
		t.Fatal("expected cached entry with minimum TTL enforcement")
	}
}

func TestCacheMinTTLUsesSmallest(t *testing.T) {
	c := NewCache()

	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 500, ParsedData: "1.2.3.4"},
		{Name: "example.com", Type: TypeA, TTL: 100, ParsedData: "5.6.7.8"},
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "9.10.11.12"},
	}
	c.Put("example.com", TypeA, records)

	// The cache uses the minimum TTL from records (100 in this case).
	c.mu.RLock()
	key := cacheKey("example.com", TypeA)
	entry := c.entries[key]
	c.mu.RUnlock()

	// The expiry should be roughly now + 100 seconds (within a tolerance).
	expectedExpiry := time.Now().Add(100 * time.Second)
	diff := entry.ExpiresAt.Sub(expectedExpiry)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expiry differs from expected by %v (entry expires at %v)", diff, entry.ExpiresAt)
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := NewCache()

	records1 := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "1.1.1.1"},
	}
	c.Put("example.com", TypeA, records1)

	records2 := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "2.2.2.2"},
	}
	c.Put("example.com", TypeA, records2)

	got := c.Get("example.com", TypeA)
	if got == nil || len(got) != 1 {
		t.Fatal("expected 1 record after overwrite")
	}
	if got[0].ParsedData != "2.2.2.2" {
		t.Errorf("ParsedData = %q, want %q after overwrite", got[0].ParsedData, "2.2.2.2")
	}
}

func TestCacheReturnsCopy(t *testing.T) {
	c := NewCache()

	records := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "1.1.1.1"},
	}
	c.Put("example.com", TypeA, records)

	got := c.Get("example.com", TypeA)
	if got == nil {
		t.Fatal("Get returned nil")
	}

	// Mutate the returned slice.
	got[0].ParsedData = "MUTATED"

	// Get again -- should not be affected by the mutation.
	got2 := c.Get("example.com", TypeA)
	if got2[0].ParsedData == "MUTATED" {
		t.Error("cache returned the same slice, not a copy")
	}
}

func TestCachePurge(t *testing.T) {
	c := NewCache()

	// Add two entries.
	records1 := []ResourceRecord{
		{Name: "fresh.com", Type: TypeA, TTL: 300, ParsedData: "1.1.1.1"},
	}
	records2 := []ResourceRecord{
		{Name: "stale.com", Type: TypeA, TTL: 300, ParsedData: "2.2.2.2"},
	}
	c.Put("fresh.com", TypeA, records1)
	c.Put("stale.com", TypeA, records2)

	// Manually expire one entry.
	c.mu.Lock()
	c.entries[cacheKey("stale.com", TypeA)].ExpiresAt = time.Now().Add(-1 * time.Second)
	c.mu.Unlock()

	c.Purge()

	// fresh.com should still be there.
	if got := c.Get("fresh.com", TypeA); got == nil {
		t.Error("Purge removed fresh entry")
	}

	// stale.com should be gone.
	// Even accessing entries directly to confirm it's been removed.
	c.mu.RLock()
	_, exists := c.entries[cacheKey("stale.com", TypeA)]
	c.mu.RUnlock()
	if exists {
		t.Error("Purge did not remove expired entry")
	}
}

func TestCachePurgeEmpty(t *testing.T) {
	c := NewCache()
	// Purging an empty cache should not panic.
	c.Purge()
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name  string
		qtype uint16
		want  string
	}{
		{"example.com", TypeA, "example.com:A"},
		{"example.com", TypeAAAA, "example.com:AAAA"},
		{"test.org", TypeMX, "test.org:MX"},
	}

	for _, tt := range tests {
		got := cacheKey(tt.name, tt.qtype)
		if got != tt.want {
			t.Errorf("cacheKey(%q, %d) = %q, want %q", tt.name, tt.qtype, got, tt.want)
		}
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := NewCache()

	var wg sync.WaitGroup
	// Run concurrent reads and writes to verify no races.
	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			records := []ResourceRecord{
				{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "1.1.1.1"},
			}
			c.Put("example.com", TypeA, records)
		}(i)

		go func(i int) {
			defer wg.Done()
			c.Get("example.com", TypeA)
		}(i)
	}

	// Also purge concurrently.
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Purge()
	}()

	wg.Wait()
}

func TestCacheMultipleTypes(t *testing.T) {
	c := NewCache()

	aRecords := []ResourceRecord{
		{Name: "example.com", Type: TypeA, TTL: 300, ParsedData: "1.2.3.4"},
	}
	aaaaRecords := []ResourceRecord{
		{Name: "example.com", Type: TypeAAAA, TTL: 300, ParsedData: "::1"},
	}

	c.Put("example.com", TypeA, aRecords)
	c.Put("example.com", TypeAAAA, aaaaRecords)

	gotA := c.Get("example.com", TypeA)
	gotAAAA := c.Get("example.com", TypeAAAA)

	if gotA == nil || gotA[0].ParsedData != "1.2.3.4" {
		t.Errorf("A record = %v, want 1.2.3.4", gotA)
	}
	if gotAAAA == nil || gotAAAA[0].ParsedData != "::1" {
		t.Errorf("AAAA record = %v, want ::1", gotAAAA)
	}
}
