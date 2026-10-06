package ledger

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrPolicyMissing       = errors.New("issuance limits must be configured first")
	ErrPolicyExceeded      = errors.New("issuance limit exceeded")
	ErrIdempotencyConflict = errors.New("idempotency key was already used for another operation")
	ErrStudentNotFound     = errors.New("student account not found")
	ErrAccountNotFound     = errors.New("account not found")
)

type Service struct{ db *sql.DB }

func New(database *sql.DB) *Service { return &Service{db: database} }

// Issue moves internal EC bonus units from a department issuance account to a
// student's account. It creates a balanced, append-only operation.
func (s *Service) Issue(ctx context.Context, tenantID, orgUnitID, actorID, studentID string, amount int64, key string) error {
	if amount <= 0 || tenantID == "" || orgUnitID == "" || actorID == "" || studentID == "" || key == "" {
		return errors.New("tenant, department, actor, student, positive amount and idempotency key are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin EC issuance: %w", err)
	}
	defer tx.Rollback()

	userAccount := "user:" + studentID
	departmentAccount := "department-issuance:" + orgUnitID
	if duplicate, err := checkDuplicate(ctx, tx, tenantID, key, studentID, amount, userAccount); err != nil {
		return err
	} else if duplicate {
		return nil
	}
	var role string
	err = tx.QueryRowContext(ctx, `SELECT role FROM users WHERE tenant_id = ? AND id = ?`, tenantID, studentID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrStudentNotFound
	}
	if err != nil {
		return fmt.Errorf("check EC recipient: %w", err)
	}
	if role != "STUDENT" {
		return ErrStudentNotFound
	}
	emissionLimit, err := readLimit(ctx, tx, tenantID, "monthly_emission_limit_ec:"+orgUnitID)
	if err != nil {
		return err
	}
	studentLimit, err := readLimit(ctx, tx, tenantID, "student_monthly_earning_limit_ec")
	if err != nil {
		return err
	}
	start, end := monthBounds(time.Now().UTC())
	var emitted, earned int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(-SUM(amount_ec), 0) FROM ledger_entries WHERE tenant_id = ? AND account_id = ? AND created_at >= ? AND created_at < ?`, tenantID, departmentAccount, start, end).Scan(&emitted); err != nil {
		return fmt.Errorf("read department issuance total: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_ec), 0) FROM ledger_entries WHERE tenant_id = ? AND account_id = ? AND amount_ec > 0 AND created_at >= ? AND created_at < ?`, tenantID, userAccount, start, end).Scan(&earned); err != nil {
		return fmt.Errorf("read student earnings total: %w", err)
	}
	if amount > emissionLimit-emitted || amount > studentLimit-earned {
		return ErrPolicyExceeded
	}
	return appendTransfer(ctx, tx, tenantID, actorID, key, "ISSUE", studentID, departmentAccount, userAccount, amount)
}

func (s *Service) Balance(ctx context.Context, tenantID, userID string) (int64, error) {
	var balance int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_ec), 0) FROM ledger_entries WHERE tenant_id = ? AND account_id = ?`, tenantID, "user:"+userID).Scan(&balance)
	return balance, err
}

// EnsureDemoWelcomeBalance grants each local demo account 10,000 EC exactly
// once. Call only from the development seed path, never from production flows.
func (s *Service) EnsureDemoWelcomeBalance(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" || userID == "" {
		return errors.New("tenant and demo account are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin demo EC grant: %w", err)
	}
	defer tx.Rollback()
	const amount = int64(10000)
	const keyPrefix = "demo-welcome-10000-v1:"
	key, account := keyPrefix+userID, "user:"+userID
	if duplicate, err := checkDuplicate(ctx, tx, tenantID, key, userID, amount, account); err != nil {
		return err
	} else if duplicate {
		return nil
	}
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM users WHERE tenant_id = ? AND id = ?`, tenantID, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return ErrAccountNotFound
	} else if err != nil {
		return fmt.Errorf("check demo account: %w", err)
	}
	return appendTransfer(ctx, tx, tenantID, "system:demo-welcome", key, "ISSUE", userID, "demo-welcome:university", account, amount)
}

func appendTransfer(ctx context.Context, tx *sql.Tx, tenantID, actorID, key, kind, referenceID, source, destination string, amount int64) error {
	operationID, err := id()
	if err != nil {
		return err
	}
	debitID, err := id()
	if err != nil {
		return err
	}
	creditID, err := id()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_operations (id, tenant_id, idempotency_key, kind, reference_id, actor_id) VALUES (?, ?, ?, ?, ?, ?)`, operationID, tenantID, key, kind, referenceID, actorID); err != nil {
		return fmt.Errorf("create EC operation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries (id, tenant_id, operation_id, account_id, amount_ec) VALUES (?, ?, ?, ?, ?)`, debitID, tenantID, operationID, source, -amount); err != nil {
		return fmt.Errorf("record EC source entry: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries (id, tenant_id, operation_id, account_id, amount_ec) VALUES (?, ?, ?, ?, ?)`, creditID, tenantID, operationID, destination, amount); err != nil {
		return fmt.Errorf("record EC destination entry: %w", err)
	}
	return tx.Commit()
}

func checkDuplicate(ctx context.Context, tx *sql.Tx, tenantID, key, studentID string, amount int64, destination string) (bool, error) {
	var operationID, kind, reference string
	err := tx.QueryRowContext(ctx, `SELECT id, kind, reference_id FROM ledger_operations WHERE tenant_id = ? AND idempotency_key = ?`, tenantID, key).Scan(&operationID, &kind, &reference)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check EC idempotency key: %w", err)
	}
	var existingAmount int64
	err = tx.QueryRowContext(ctx, `SELECT amount_ec FROM ledger_entries WHERE tenant_id = ? AND operation_id = ? AND account_id = ?`, tenantID, operationID, destination).Scan(&existingAmount)
	if err != nil {
		return false, fmt.Errorf("check existing EC operation: %w", err)
	}
	if kind != "ISSUE" || reference != studentID || existingAmount != amount {
		return false, ErrIdempotencyConflict
	}
	return true, nil
}

func readLimit(ctx context.Context, tx *sql.Tx, tenantID, key string) (int64, error) {
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT rule_value FROM tenant_rules WHERE tenant_id = ? AND rule_key = ?`, tenantID, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrPolicyMissing
	}
	if err != nil {
		return 0, fmt.Errorf("read EC policy: %w", err)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, ErrPolicyMissing
	}
	return value, nil
}

func monthBounds(now time.Time) (string, string) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	const layout = "2006-01-02 15:04:05"
	return start.Format(layout), end.Format(layout)
}

func id() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
