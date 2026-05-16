package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir for %q: %w", path, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Store) ListConnections() ([]connection.Record, error) {
	rows, err := s.db.Query(`
		SELECT id, provider_id, api_key, access_token, refresh_token, token_type, expires_in, access_token_expires_at, name
		FROM connections
		ORDER BY provider_id, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var out []connection.Record
	for rows.Next() {
		var item connection.Record
		if err := rows.Scan(
			&item.ID,
			&item.ProviderID,
			&item.APIKey,
			&item.AccessToken,
			&item.RefreshToken,
			&item.TokenType,
			&item.ExpiresIn,
			&item.AccessTokenExpiresAt,
			&item.Name,
		); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connections: %w", err)
	}

	return out, nil
}

func (s *Store) GetConnection(id string) (connection.Record, bool, error) {
	var item connection.Record
	err := s.db.QueryRow(`
		SELECT id, provider_id, api_key, access_token, refresh_token, token_type, expires_in, access_token_expires_at, name
		FROM connections
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.ProviderID,
		&item.APIKey,
		&item.AccessToken,
		&item.RefreshToken,
		&item.TokenType,
		&item.ExpiresIn,
		&item.AccessTokenExpiresAt,
		&item.Name,
	)
	if err == sql.ErrNoRows {
		return connection.Record{}, false, nil
	}
	if err != nil {
		return connection.Record{}, false, fmt.Errorf("get connection %q: %w", id, err)
	}

	return item, true, nil
}

func (s *Store) CreateConnection(item connection.Record) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO connections (
			id, provider_id, api_key, access_token, refresh_token, token_type, expires_in, access_token_expires_at, name, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ProviderID, item.APIKey, item.AccessToken, item.RefreshToken, item.TokenType, item.ExpiresIn, item.AccessTokenExpiresAt, item.Name, now, now)
	if err != nil {
		return fmt.Errorf("create connection %q: %w", item.ID, err)
	}

	return nil
}

func (s *Store) UpdateConnection(previousID string, item connection.Record) error {
	result, err := s.db.Exec(`
		UPDATE connections
		SET id = ?, provider_id = ?, api_key = ?, access_token = ?, refresh_token = ?, token_type = ?, expires_in = ?, access_token_expires_at = ?, name = ?, updated_at = ?
		WHERE id = ?
	`, item.ID, item.ProviderID, item.APIKey, item.AccessToken, item.RefreshToken, item.TokenType, item.ExpiresIn, item.AccessTokenExpiresAt, item.Name, time.Now().Unix(), previousID)
	if err != nil {
		return fmt.Errorf("update connection %q: %w", previousID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated rows for %q: %w", previousID, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) DeleteConnection(id string) error {
	result, err := s.db.Exec(`DELETE FROM connections WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete connection %q: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted rows for %q: %w", id, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) ReplaceConnections(items []connection.Record) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin connection replacement: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM connections`); err != nil {
		return fmt.Errorf("clear connections: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO connections (
			id, provider_id, api_key, access_token, refresh_token, token_type, expires_in, access_token_expires_at, name, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare connection replacement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.ProviderID, item.APIKey, item.AccessToken, item.RefreshToken, item.TokenType, item.ExpiresIn, item.AccessTokenExpiresAt, item.Name, now, now); err != nil {
			return fmt.Errorf("replace connection %q: %w", item.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit connection replacement: %w", err)
	}

	return nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY,
			provider_id TEXT NOT NULL,
			api_key TEXT NOT NULL DEFAULT '',
			access_token TEXT NOT NULL DEFAULT '',
			refresh_token TEXT NOT NULL DEFAULT '',
			token_type TEXT NOT NULL DEFAULT '',
			expires_in INTEGER NOT NULL DEFAULT 0,
			access_token_expires_at INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}

	return nil
}
