package m_lib_db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresLib struct {
	DB_lib *sql.DB
}

func NewLibPostgres(db_lib_url string, smoc, smic int) (*PostgresLib, error) {
	db, err := sql.Open("pgx", db_lib_url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(smoc)
	db.SetMaxIdleConns(smic)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	p := &PostgresLib{DB_lib: db}
	if err := p.ensureSchema(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PostgresLib) Close() error {
	return p.DB_lib.Close()
}

func (p *PostgresLib) ensureSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	schrema := `
	CREATE TABLE IF NOT EXISTS lib (
		id BIGINT NOT NULL,
		course TEXT NOT NULL,
		name TEXT NOT NULL,
		cond TEXT NOT NULL,
		data TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		PRIMARY KEY (course, name)
	);`
	_, err := p.DB_lib.ExecContext(ctx, schrema)
	return err
}

func (p *PostgresLib) InsertData(ctx context.Context, id int64, course, name, cond, data string) error {
	_, err := p.DB_lib.ExecContext(
		ctx,
		"INSERT INTO lib (id, course, name, cond, data) VALUES ($1,$2,$3,$4,$5)",
		id,
		course,
		name,
		cond,
		data,
	)
	return err
}

func (p *PostgresLib) GetData(ctx context.Context, course, name string) (string, string, string, error) {
	var (
		course_db string
		name_db   string
		cond      string
		data      string
	)
	row := p.DB_lib.QueryRowContext(
		ctx,
		"SELECT course, name, cond, data FROM lib WHERE course=$1 AND name=$2",
		course,
		name,
	)
	if err := row.Scan(&course_db, &name_db, &cond, &data); err != nil {
		return "", "", "", err
	}
	return name_db, cond, data, nil
}

func (p *PostgresLib) UpdateData(ctx context.Context, course, name, cond, data string) error {
	_, err := p.DB_lib.ExecContext(
		ctx,
		"UPDATE lib SET cond=$1, data=$2 WHERE course=$3 and name=$4",
		cond,
		data,
		course,
		name,
	)
	return err
}

func (p *PostgresLib) DeleteData(ctx context.Context, code string, id int64) error {
	_, err := p.DB_lib.ExecContext(
		ctx,
		"DELETE FROM lib WHERE data=$1 AND id=$2",
		code,
		id,
	)
	return err
}
