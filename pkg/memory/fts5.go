package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

var (
	nonWordChars = regexp.MustCompile(`[^\p{L}\p{N}_\-]+`)
)

// Storage handles persistent storage of browser knowledge using SQLite FTS5.
type Storage struct {
	db *sql.DB
}

// OpenStorage initializes SQLite with WAL mode for ultra-fast concurrent reads.
func OpenStorage(dbPath string) (*Storage, error) {
	if dbPath == "" {
		dbPath = filepath.Join(".ai-browser-memory", "browser_memory.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return &Storage{db: db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS playbooks (
		domain TEXT PRIMARY KEY,
		data_json TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS handoffs (
		id TEXT PRIMARY KEY,
		data_json TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
		url,
		title,
		domain,
		content,
		tokenize = 'unicode61 remove_diacritics 2'
	);
	`
	_, err := db.Exec(schema)
	return err
}

// Close closes the underlying database.
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SavePlaybook persists a domain playbook.
func (s *Storage) SavePlaybook(pb *DomainPlaybook) error {
	if pb == nil || pb.Domain == "" {
		return fmt.Errorf("invalid playbook")
	}
	pb.UpdatedAt = time.Now()

	data, err := json.Marshal(pb)
	if err != nil {
		return err
	}

	query := `INSERT INTO playbooks (domain, data_json, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET data_json = excluded.data_json, updated_at = excluded.updated_at`
	_, err = s.db.Exec(query, pb.Domain, string(data), pb.UpdatedAt.Unix())
	return err
}

// GetPlaybook loads a domain playbook. Returns nil, nil if not found.
func (s *Storage) GetPlaybook(domain string) (*DomainPlaybook, error) {
	var dataStr string
	query := `SELECT data_json FROM playbooks WHERE domain = ?`
	err := s.db.QueryRow(query, domain).Scan(&dataStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pb DomainPlaybook
	if err := json.Unmarshal([]byte(dataStr), &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// IndexPage stores or updates a visited page into the FTS5 full-text index.
func (s *Storage) IndexPage(rec PageRecord) error {
	if rec.URL == "" {
		return fmt.Errorf("url cannot be empty")
	}

	// Delete prior entry for this URL if existing
	_, _ = s.db.Exec(`DELETE FROM pages_fts WHERE url = ?`, rec.URL)

	// Clean content to keep index compact
	cleanContent := strings.TrimSpace(rec.Content)
	if len(cleanContent) > 200000 {
		cleanContent = cleanContent[:200000] // Cap to ~200KB per page
	}

	query := `INSERT INTO pages_fts (url, title, domain, content) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, rec.URL, rec.Title, rec.Domain, cleanContent)
	return err
}

// SearchPages executes an FTS5 full-text search with BM25 ranking and snippet extraction.
func (s *Storage) SearchPages(query string, limit int) ([]PageRecord, error) {
	if limit <= 0 {
		limit = 5
	}

	sanitized := sanitizeFTS5Query(query)
	if sanitized == "" {
		return nil, nil
	}

	sqlQuery := `
		SELECT url, title, domain, snippet(pages_fts, 3, '<b>', '</b>', '...', 24), bm25(pages_fts) as rank
		FROM pages_fts
		WHERE pages_fts MATCH ?
		ORDER BY rank ASC
		LIMIT ?
	`
	rows, err := s.db.Query(sqlQuery, sanitized, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PageRecord
	for rows.Next() {
		var rec PageRecord
		if err := rows.Scan(&rec.URL, &rec.Title, &rec.Domain, &rec.Snippet, &rec.Rank); err != nil {
			continue
		}
		results = append(results, rec)
	}
	return results, nil
}

// SaveHandoff stores a cross-agent handoff session.
func (s *Storage) SaveHandoff(h *HandoffSession) error {
	if h == nil || h.ID == "" {
		return fmt.Errorf("invalid handoff")
	}
	h.SavedAt = time.Now()

	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	query := `INSERT INTO handoffs (id, data_json, created_at) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET data_json = excluded.data_json, created_at = excluded.created_at`
	_, err = s.db.Exec(query, h.ID, string(data), h.SavedAt.Unix())
	return err
}

// GetHandoff retrieves a saved handoff session.
func (s *Storage) GetHandoff(id string) (*HandoffSession, error) {
	var dataStr string
	query := `SELECT data_json FROM handoffs WHERE id = ?`
	err := s.db.QueryRow(query, id).Scan(&dataStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var h HandoffSession
	if err := json.Unmarshal([]byte(dataStr), &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func sanitizeFTS5Query(q string) string {
	tokens := nonWordChars.Split(strings.TrimSpace(q), -1)
	var valid []string
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if len(t) > 1 {
			valid = append(valid, fmt.Sprintf(`"%s"`, t))
		}
	}
	if len(valid) == 0 {
		return ""
	}
	return strings.Join(valid, " OR ")
}
