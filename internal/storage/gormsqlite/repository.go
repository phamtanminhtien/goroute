package gormsqlite

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func Open(path string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir for %q: %w", path, err)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}

	repo := &Repository{db: db}
	if err := repo.Migrate(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}

	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("resolve database handle: %w", err)
	}

	return sqlDB.Close()
}
