package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages request history persistence.
type Store struct {
	db *sql.DB
}

// NewStore creates a new history store at the given path.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening history db: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			method        TEXT NOT NULL,
			url           TEXT NOT NULL,
			status_code   INTEGER,
			duration_ns   INTEGER,
			size          INTEGER,
			request_body  TEXT,
			response_body TEXT,
			headers       TEXT,
			timestamp     TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_history_url ON history(url);
	`)
	if err != nil {
		return fmt.Errorf("creating history table: %w", err)
	}
	return nil
}

// Add inserts a new history entry.
func (s *Store) Add(e Entry) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO history (method, url, status_code, duration_ns, size, request_body, response_body, headers, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Method, e.URL, e.StatusCode, e.Duration.Nanoseconds(), e.Size,
		e.RequestBody, e.ResponseBody, e.Headers,
		e.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("inserting history: %w", err)
	}
	return result.LastInsertId()
}

// List returns the most recent entries.
func (s *Store) List(limit, offset int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, method, url, status_code, duration_ns, size, request_body, response_body, headers, timestamp
		FROM history
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing history: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Search searches history by URL substring.
func (s *Store) Search(query string) ([]Entry, error) {
	rows, err := s.db.Query(`
		SELECT id, method, url, status_code, duration_ns, size, request_body, response_body, headers, timestamp
		FROM history
		WHERE url LIKE ?
		ORDER BY timestamp DESC
		LIMIT 50`, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("searching history: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Clear removes all history entries.
func (s *Store) Clear() error {
	_, err := s.db.Exec("DELETE FROM history")
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var e Entry
		var durationNs int64
		var ts string
		err := rows.Scan(&e.ID, &e.Method, &e.URL, &e.StatusCode, &durationNs,
			&e.Size, &e.RequestBody, &e.ResponseBody, &e.Headers, &ts)
		if err != nil {
			return nil, fmt.Errorf("scanning history row: %w", err)
		}
		e.Duration = time.Duration(durationNs)
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
