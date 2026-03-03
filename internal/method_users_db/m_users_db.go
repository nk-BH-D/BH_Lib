package m_us_db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresUs struct {
	DB_us *sql.DB
}

func NewUsPostgres(db_us_url string, smoc, smic int) (*PostgresUs, error) {
	db, err := sql.Open("pgx", db_us_url)
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

	p := &PostgresUs{DB_us: db}
	if err := p.ensureSchema(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PostgresUs) Close() error {
	return p.DB_us.Close()
}

func (p *PostgresUs) ensureSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	schrema := `
	CREATE TABLE IF NOT EXISTS users (
		user_id BIGINT PRIMARY KEY,
		chat_id BIGINT,
		login TEXT UNIQUE,
		status TEXT NOT NULL,
		password TEXT,
		quantity INTEGER NOT NULL,
		request_name TEXT,
		creared_ad TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
	_, err := p.DB_us.ExecContext(ctx, schrema)
	return err
}

func (p *PostgresUs) InsertUser(ctx context.Context, user_id, chat_id int64, status, password, login, request_name string, quantity int32) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"INSERT INTO users (user_id, chat_id, login, status, password, quantity, request_name) VALUES ($1,$2,$3,$4,$5,$6,$7)",
		user_id,
		chat_id,
		login,
		status,
		password,
		quantity,
		request_name,
	)
	return err
}

func (p *PostgresUs) GetUserStatus(ctx context.Context, user_id int64) (string, string, string, error) {
	var (
		login    string
		status   string
		password string
	)
	row := p.DB_us.QueryRowContext(
		ctx,
		"SELECT login, status, password FROM users WHERE user_id=$1",
		user_id,
	)
	if err := row.Scan(&login, &status, &password); err != nil {
		return "", "", "", err
	}
	return login, status, password, nil
}

func (p *PostgresUs) GetAdminAndRootRequests(ctx context.Context) (string, error) {
	query := `
        SELECT login, request_name
        FROM users
        WHERE status IN ('admin', 'root')
        ORDER BY login
    `

	rows, err := p.DB_us.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var result strings.Builder
	totalRequests := 0 // Счётчик подходящих запросов

	for rows.Next() {
		var login string
		var requestNames sql.NullString

		if err := rows.Scan(&login, &requestNames); err != nil {
			return "", err
		}

		if requestNames.String == "" {
			return "", nil
		}

		// Добавляем логин в результат
		result.WriteString(fmt.Sprintf("%s\n", login))

		// Если у пользователя есть запросы, обрабатываем их
		if requestNames.Valid {
			requestList := strings.Split(requestNames.String, "\n")
			for _, request := range requestList {
				request = strings.TrimSpace(request)
				// Пропускаем запросы, которые начинаются с "/"
				if !strings.HasPrefix(request, "/") && request != "" {
					result.WriteString(fmt.Sprintf("  %s\n", request))
					totalRequests++ // Увеличиваем счётчик подходящих запросов
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	// Добавляем строку с общим количеством запросов в конец результата
	result.WriteString(fmt.Sprintf("\nВсего запросов: %d\n", totalRequests))

	return result.String(), nil
}

func (p *PostgresUs) GetUserRequestnames(ctx context.Context, user_id int64) (string, int32, error) {
	var (
		request_name string
		quantity     int32
	)
	row := p.DB_us.QueryRowContext(
		ctx,
		"SELECT request_name, quantity FROM users WHERE user_id=$1",
		user_id,
	)
	if err := row.Scan(&request_name, &quantity); err != nil {
		return "", 0, err
	}
	return request_name, quantity, nil
}

func (p *PostgresUs) UpdateUserStatus(ctx context.Context, user_id int64, status, password string) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET status=$1, password=$2 WHERE user_id=$3",
		status,
		password,
		user_id,
	)
	return err
}

func (p *PostgresUs) UpdateUserData(ctx context.Context, user_id int64, login, request_name string, quantity int32) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET login=$1, quantity=quantity+$2, request_name=request_name || '\n' || $3 WHERE user_id=$4",
		login,
		quantity,
		request_name,
		user_id,
	)
	return err
}

func (p *PostgresUs) DeleteReqData(ctx context.Context, del string, user_id int64) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		`UPDATE users 
		SET request_name = REPLACE(
				REPLACE(request_name, CONCAT('c', $1::text), ''),
				CONCAT('up', $1::text), ''
			), 
			quantity=quantity-1 
		WHERE 
			user_id=$2 
			AND (
				request_name LIKE CONCAT('%', 'c', $1::text, '%')
				OR
				request_name LIKE CONCAT('%', 'up', $1::text, '%')
			)		
		`,
		del,
		user_id,
	)
	return err
}

func (p *PostgresUs) UpUserData(ctx context.Context, user_id int64, up string) error {
	del := "c" + up
	upd := "up" + up
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET request_name = REPLACE(request_name, $1, $2) WHERE user_id=$3 AND request_name LIKE '%' || $1 || '%'",
		del,
		upd,
		user_id,
	)
	return err
}

func (p *PostgresUs) ChangeUserStatus(ctx context.Context, user_id int64, status, password string) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET status=$1, password=$2 WHERE user_id=$3",
		status,
		password,
		user_id,
	)
	return err
}
