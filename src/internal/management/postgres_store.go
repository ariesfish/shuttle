package management

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const postgresStateKey = "management-store"

type PostgresOptions struct {
	DSN string
}

func NewPostgresStore(ctx context.Context, options PostgresOptions) (*FileStore, error) {
	db, err := sql.Open("pgx", options.DSN)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := initPostgresSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	data := newStoreData()
	var raw []byte
	err = db.QueryRowContext(ctx, `SELECT data FROM platform_state WHERE key = $1`, postgresStateKey).Scan(&raw)
	if err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			_ = db.Close()
			return nil, err
		}
	} else if err != nil && err != sql.ErrNoRows {
		_ = db.Close()
		return nil, err
	}
	normalizeStoreData(&data)

	store := &FileStore{data: data, now: time.Now, recipes: MustLoadDefaultRecipeRegistry()}
	store.persist = func(data storeData) error {
		contents, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = db.ExecContext(context.Background(), `
			INSERT INTO platform_state (key, data, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (key) DO UPDATE SET data = EXCLUDED.data, updated_at = now()
		`, postgresStateKey, contents)
		return err
	}
	return store, nil
}

func initPostgresSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS platform_state (
			key text PRIMARY KEY,
			data jsonb NOT NULL,
			updated_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	return err
}

func normalizeStoreData(data *storeData) {
	if data.NextID == 0 {
		data.NextID = 1
	}
	if data.Projects == nil {
		data.Projects = map[string]Project{}
	}
	if data.Clusters == nil {
		data.Clusters = map[string]InferenceCluster{}
	}
	if data.Agents == nil {
		data.Agents = map[string]ClusterAgent{}
	}
	if data.ModelArtifacts == nil {
		data.ModelArtifacts = map[string]ModelArtifact{}
	}
	if data.ServingApplications == nil {
		data.ServingApplications = map[string]ServingApplication{}
	}
	if data.Transitions == nil {
		data.Transitions = map[string]ServingApplicationTransition{}
	}
	if data.Endpoints == nil {
		data.Endpoints = map[string]EndpointRegistryEntry{}
	}
	if data.AuditRecords == nil {
		data.AuditRecords = map[string]AuditRecord{}
	}
	if data.Tasks == nil {
		data.Tasks = map[string]Task{}
	}
}
