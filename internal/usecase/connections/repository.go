package connections

import "github.com/phamtanminhtien/goroute/internal/domain/connection"

type Repository interface {
	ListConnections() ([]connection.Record, error)
	GetConnection(string) (connection.Record, bool, error)
	CreateConnection(connection.Record) error
	UpdateConnection(string, connection.Record) error
	SetProviderConnectionsEnabled(string, bool) ([]connection.Record, error)
	RecordConnectionRuntimeError(id string, message string, category string, lastErrorAt int64, retryAfter int64) error
	ClearConnectionRuntimeError(id string) error
	DeleteConnection(string) error
	ReplaceConnections([]connection.Record) error
	Close() error
}
