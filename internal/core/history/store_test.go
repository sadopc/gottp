package history

import (
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Add entries
	id1, err := store.Add(Entry{
		Method:     "GET",
		URL:        "https://api.example.com/users",
		StatusCode: 200,
		Duration:   150 * time.Millisecond,
		Size:       1024,
		Timestamp:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if id1 == 0 {
		t.Error("expected non-zero ID")
	}

	id2, err := store.Add(Entry{
		Method:       "POST",
		URL:          "https://api.example.com/users",
		StatusCode:   201,
		Duration:     200 * time.Millisecond,
		Size:         512,
		RequestBody:  `{"name":"test"}`,
		ResponseBody: `{"id":1}`,
		Timestamp:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// List
	entries, err := store.List(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Most recent first
	if entries[0].ID != id2 {
		t.Errorf("expected most recent first, got id %d", entries[0].ID)
	}

	// Search
	results, err := store.Search("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 search results, got %d", len(results))
	}

	results, err = store.Search("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	// Clear
	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	entries, err = store.List(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestStore_ListFiltered(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now()
	store.Add(Entry{Method: "GET", URL: "https://api.example.com/users", StatusCode: 200, Timestamp: now.Add(-3 * time.Hour)})
	store.Add(Entry{Method: "POST", URL: "https://api.example.com/users", StatusCode: 201, Timestamp: now.Add(-2 * time.Hour)})
	store.Add(Entry{Method: "GET", URL: "https://other.com/data", StatusCode: 404, Timestamp: now.Add(-1 * time.Hour)})
	store.Add(Entry{Method: "DELETE", URL: "https://api.example.com/users/1", StatusCode: 500, Timestamp: now})

	// Filter by method
	entries, err := store.ListFiltered(Filter{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 GET entries, got %d", len(entries))
	}

	// Filter by status code
	entries, err = store.ListFiltered(Filter{StatusCode: 404})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry with status 404, got %d", len(entries))
	}

	// Filter by status range
	entries, err = store.ListFiltered(Filter{StatusMin: 200, StatusMax: 299})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with 2xx status, got %d", len(entries))
	}

	// Filter by URL pattern
	entries, err = store.ListFiltered(Filter{URLPattern: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries matching example.com, got %d", len(entries))
	}

	// Filter by time range
	entries, err = store.ListFiltered(Filter{Since: now.Add(-90 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(entries))
	}
}

func TestStore_CountAndDelete(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	id1, _ := store.Add(Entry{Method: "GET", URL: "https://example.com", Timestamp: time.Now()})
	store.Add(Entry{Method: "POST", URL: "https://example.com", Timestamp: time.Now()})

	count, err := store.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	if err := store.Delete(id1); err != nil {
		t.Fatal(err)
	}

	count, err = store.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected count 1 after delete, got %d", count)
	}
}

func TestStore_DurationRoundTrip(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dur := 123456789 * time.Nanosecond
	_, err = store.Add(Entry{
		Method:    "GET",
		URL:       "https://example.com",
		Duration:  dur,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := store.List(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Duration != dur {
		t.Errorf("duration mismatch: got %v, want %v", entries[0].Duration, dur)
	}
}
