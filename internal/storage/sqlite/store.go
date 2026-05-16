package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
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

func (s *Store) CreateRequestAttemptHistory(record chatcompletion.RequestAttemptHistory) (chatcompletion.RequestAttemptHistory, error) {
	attempts, err := json.Marshal(record.Attempts)
	if err != nil {
		return chatcompletion.RequestAttemptHistory{}, fmt.Errorf("encode request history attempts: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO request_history (
			request_id, request_path, requested_model, resolved_target, provider_id, provider_name, stream, status, final_status, final_error_category, last_error_category, last_error_message, message_count, tool_count, attempt_count, last_connection_id, last_connection_name, started_at, last_attempt_at, completed_at, updated_at, attempts_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.RequestID,
		record.RequestPath,
		record.RequestedModel,
		record.ResolvedTarget,
		record.ProviderID,
		record.ProviderName,
		boolToInt(record.Stream),
		record.Status,
		record.FinalStatus,
		record.FinalErrorCategory,
		record.LastErrorCategory,
		record.LastErrorMessage,
		record.MessageCount,
		record.ToolCount,
		record.AttemptCount,
		record.LastConnectionID,
		record.LastConnectionName,
		record.StartedAt.UTC().UnixMilli(),
		timeToUnixMilli(record.LastAttemptAt),
		timeToUnixMilli(record.CompletedAt),
		timeToUnixMilli(record.UpdatedAt),
		string(attempts),
	)
	if err != nil {
		return chatcompletion.RequestAttemptHistory{}, fmt.Errorf("insert request history: %w", err)
	}

	historyID, err := result.LastInsertId()
	if err != nil {
		return chatcompletion.RequestAttemptHistory{}, fmt.Errorf("read request history id: %w", err)
	}

	record.HistoryID = historyID
	return record, nil
}

func (s *Store) UpdateRequestAttemptHistory(record chatcompletion.RequestAttemptHistory) error {
	attempts, err := json.Marshal(record.Attempts)
	if err != nil {
		return fmt.Errorf("encode request history attempts: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE request_history
		SET request_id = ?, request_path = ?, requested_model = ?, resolved_target = ?, provider_id = ?, provider_name = ?, stream = ?, status = ?, final_status = ?, final_error_category = ?, last_error_category = ?, last_error_message = ?, message_count = ?, tool_count = ?, attempt_count = ?, last_connection_id = ?, last_connection_name = ?, started_at = ?, last_attempt_at = ?, completed_at = ?, updated_at = ?, attempts_json = ?
		WHERE id = ?
	`,
		record.RequestID,
		record.RequestPath,
		record.RequestedModel,
		record.ResolvedTarget,
		record.ProviderID,
		record.ProviderName,
		boolToInt(record.Stream),
		record.Status,
		record.FinalStatus,
		record.FinalErrorCategory,
		record.LastErrorCategory,
		record.LastErrorMessage,
		record.MessageCount,
		record.ToolCount,
		record.AttemptCount,
		record.LastConnectionID,
		record.LastConnectionName,
		record.StartedAt.UTC().UnixMilli(),
		timeToUnixMilli(record.LastAttemptAt),
		timeToUnixMilli(record.CompletedAt),
		record.UpdatedAt.UTC().UnixMilli(),
		string(attempts),
		record.HistoryID,
	)
	if err != nil {
		return fmt.Errorf("update request history %d: %w", record.HistoryID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated request history rows for %d: %w", record.HistoryID, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) RecentRequestAttempts(limit int) ([]chatcompletion.RequestAttemptHistory, error) {
	if limit <= 0 {
		limit = 1
	}

	rows, err := s.db.Query(`
		SELECT id, request_id, request_path, requested_model, resolved_target, provider_id, provider_name, stream, status, final_status, final_error_category, last_error_category, last_error_message, message_count, tool_count, attempt_count, last_connection_id, last_connection_name, started_at, last_attempt_at, completed_at, updated_at, attempts_json
		FROM request_history
		ORDER BY updated_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list request history: %w", err)
	}
	defer rows.Close()

	var out []chatcompletion.RequestAttemptHistory
	for rows.Next() {
		var item chatcompletion.RequestAttemptHistory
		var stream int
		var startedAt int64
		var lastAttemptAt int64
		var completedAt int64
		var updatedAt int64
		var attemptsJSON string
		if err := rows.Scan(
			&item.HistoryID,
			&item.RequestID,
			&item.RequestPath,
			&item.RequestedModel,
			&item.ResolvedTarget,
			&item.ProviderID,
			&item.ProviderName,
			&stream,
			&item.Status,
			&item.FinalStatus,
			&item.FinalErrorCategory,
			&item.LastErrorCategory,
			&item.LastErrorMessage,
			&item.MessageCount,
			&item.ToolCount,
			&item.AttemptCount,
			&item.LastConnectionID,
			&item.LastConnectionName,
			&startedAt,
			&lastAttemptAt,
			&completedAt,
			&updatedAt,
			&attemptsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan request history: %w", err)
		}

		item.Stream = stream != 0
		item.StartedAt = unixMilliToTime(startedAt)
		item.LastAttemptAt = unixMilliToTime(lastAttemptAt)
		item.CompletedAt = unixMilliToTime(completedAt)
		item.UpdatedAt = unixMilliToTime(updatedAt)
		if err := json.Unmarshal([]byte(attemptsJSON), &item.Attempts); err != nil {
			return nil, fmt.Errorf("decode request history attempts: %w", err)
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate request history: %w", err)
	}

	return out, nil
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
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS request_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL DEFAULT '',
			request_path TEXT NOT NULL DEFAULT '/v1/chat/completions',
			requested_model TEXT NOT NULL,
			resolved_target TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			provider_name TEXT NOT NULL,
			stream INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'started',
			final_status TEXT NOT NULL DEFAULT '',
			final_error_category TEXT NOT NULL DEFAULT '',
			last_error_category TEXT NOT NULL DEFAULT '',
			last_error_message TEXT NOT NULL DEFAULT '',
			message_count INTEGER NOT NULL DEFAULT 0,
			tool_count INTEGER NOT NULL DEFAULT 0,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			last_connection_id TEXT NOT NULL DEFAULT '',
			last_connection_name TEXT NOT NULL DEFAULT '',
			started_at INTEGER NOT NULL,
			last_attempt_at INTEGER NOT NULL DEFAULT 0,
			completed_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			attempts_json TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func timeToUnixMilli(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}

	return value.UTC().UnixMilli()
}

func unixMilliToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(value).UTC()
}
