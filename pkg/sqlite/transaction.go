package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

type Transaction struct {
	tx        *sql.Tx
	mu        sync.Mutex
	completed bool
}

func (db *DB) BeginTransaction(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sqlite: begin transaction: %w", err)
	}
	return &Transaction{tx: tx}, nil
}

func (db *DB) WithTransaction(ctx context.Context, opts *sql.TxOptions, fn func(tx *Transaction) error) error {
	tx, err := db.BeginTransaction(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("sqlite: transaction failed: %w; rollback failed: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

func (tx *Transaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.completed {
		return fmt.Errorf("sqlite: transaction already completed")
	}
	tx.completed = true
	if err := tx.tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit transaction: %w", err)
	}
	return nil
}

func (tx *Transaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.completed {
		return nil
	}
	tx.completed = true
	if err := tx.tx.Rollback(); err != nil {
		return fmt.Errorf("sqlite: rollback transaction: %w", err)
	}
	return nil
}

func (tx *Transaction) ensureActive() error {
	if tx == nil || tx.tx == nil {
		return fmt.Errorf("sqlite: transaction is nil")
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.completed {
		return fmt.Errorf("sqlite: transaction already completed")
	}
	return nil
}
