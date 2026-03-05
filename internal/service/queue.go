package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type QueueService struct {
	db             *sql.DB
	avgServiceMins int
}

func NewQueueService(db *sql.DB, avgServiceMins int) *QueueService {
	return &QueueService{
		db:             db,
		avgServiceMins: avgServiceMins,
	}
}

// CreateToken creates a new token for a service (A/D/L)
// Returns token code, position in waiting queue for that service, and estimated minutes.
func (s *QueueService) CreateToken(ctx context.Context, serviceCode, customerName string) (token string, position int, estMins int, err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var serviceID int
	if err = tx.QueryRowContext(ctx, `SELECT id FROM services WHERE code=?`, serviceCode).Scan(&serviceID); err != nil {
		return "", 0, 0, fmt.Errorf("service not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert first, then create token_code using row id (simple + unique)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO tokens(service_id, token_code, customer_name, status, created_at)
		VALUES (?, '', ?, 'waiting', ?)`,
		serviceID, customerName, now,
	)
	if err != nil {
		return "", 0, 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return "", 0, 0, err
	}

	token = fmt.Sprintf("%s-%d", serviceCode, id)

	if _, err = tx.ExecContext(ctx, `UPDATE tokens SET token_code=? WHERE id=?`, token, id); err != nil {
		return "", 0, 0, err
	}

	// position in waiting queue for THIS service
	if err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tokens WHERE service_id=? AND status='waiting'`,
		serviceID,
	).Scan(&position); err != nil {
		return "", 0, 0, err
	}

	estMins = position * s.avgServiceMins

	if err = tx.Commit(); err != nil {
		return "", 0, 0, err
	}
	return token, position, estMins, nil
}

// QueueStatus returns current serving token and total waiting count (all services)
func (s *QueueService) QueueStatus(ctx context.Context) (currentToken string, waiting int, err error) {
	_ = s.db.QueryRowContext(ctx,
		`SELECT COALESCE((SELECT token_code FROM tokens WHERE status='serving' ORDER BY id DESC LIMIT 1), '')`,
	).Scan(&currentToken)

	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tokens WHERE status='waiting'`).Scan(&waiting)
	return currentToken, waiting, err
}

// Next ends current serving (if any) and moves the oldest waiting to serving.
// Returns the new serving token (or "" if none waiting).
func (s *QueueService) Next(ctx context.Context) (currentToken string, err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC().Format(time.RFC3339)

	// Mark current serving done (if exists)
	_, _ = tx.ExecContext(ctx, `UPDATE tokens SET status='done', done_at=? WHERE status='serving'`, now)

	// Pick oldest waiting token across all services
	var nextID int
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM tokens WHERE status='waiting' ORDER BY id ASC LIMIT 1`,
	).Scan(&nextID)

	if err == sql.ErrNoRows {
		// no waiting tokens
		if err = tx.Commit(); err != nil {
			return "", err
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}

	// Move it to serving
	if _, err = tx.ExecContext(ctx, `UPDATE tokens SET status='serving', served_at=? WHERE id=?`, now, nextID); err != nil {
		return "", err
	}

	// Return token code
	if err = tx.QueryRowContext(ctx, `SELECT token_code FROM tokens WHERE id=?`, nextID).Scan(&currentToken); err != nil {
		return "", err
	}

	return currentToken, tx.Commit()
}

// ListServices returns all available services (A/D/L)
func (s *QueueService) ListServices(ctx context.Context) ([]struct {
	Code string
	Name string
}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT code, name FROM services ORDER BY code ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []struct {
		Code string
		Name string
	}
	for rows.Next() {
		var item struct {
			Code string
			Name string
		}
		if err := rows.Scan(&item.Code, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
