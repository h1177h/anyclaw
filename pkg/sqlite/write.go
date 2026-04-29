package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type ExecResult struct {
	LastInsertID int64
	RowsAffected int64
}

func (db *DB) InsertRow(ctx context.Context, table string, data map[string]any) (ExecResult, error) {
	return insertRow(ctx, db, table, data)
}

func (tx *Transaction) InsertRow(ctx context.Context, table string, data map[string]any) (ExecResult, error) {
	if err := tx.ensureActive(); err != nil {
		return ExecResult{}, err
	}
	return insertRow(ctx, tx.tx, table, data)
}

func (db *DB) UpdateRows(ctx context.Context, table string, data map[string]any, filters ...Filter) (int64, error) {
	return updateRows(ctx, db, table, data, filters)
}

func (tx *Transaction) UpdateRows(ctx context.Context, table string, data map[string]any, filters ...Filter) (int64, error) {
	if err := tx.ensureActive(); err != nil {
		return 0, err
	}
	return updateRows(ctx, tx.tx, table, data, filters)
}

func (db *DB) UpsertRow(ctx context.Context, table string, data map[string]any, conflictColumns []string) (ExecResult, error) {
	return upsertRow(ctx, db, table, data, conflictColumns)
}

func (tx *Transaction) UpsertRow(ctx context.Context, table string, data map[string]any, conflictColumns []string) (ExecResult, error) {
	if err := tx.ensureActive(); err != nil {
		return ExecResult{}, err
	}
	return upsertRow(ctx, tx.tx, table, data, conflictColumns)
}

func (db *DB) DeleteRows(ctx context.Context, table string, filters ...Filter) (int64, error) {
	return deleteRows(ctx, db, table, filters)
}

func (tx *Transaction) DeleteRows(ctx context.Context, table string, filters ...Filter) (int64, error) {
	if err := tx.ensureActive(); err != nil {
		return 0, err
	}
	return deleteRows(ctx, tx.tx, table, filters)
}

func insertRow(ctx context.Context, exec sqlExecutor, table string, data map[string]any) (ExecResult, error) {
	if len(data) == 0 {
		return ExecResult{}, fmt.Errorf("sqlite: insert row: no data provided")
	}
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return ExecResult{}, err
	}

	columns := sortedKeys(data)
	columnSQL := make([]string, 0, len(columns))
	placeholders := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return ExecResult{}, err
		}
		columnSQL = append(columnSQL, quoted)
		placeholders = append(placeholders, "?")
		args = append(args, data[column])
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableSQL, strings.Join(columnSQL, ", "), strings.Join(placeholders, ", "))
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return ExecResult{}, fmt.Errorf("sqlite: insert row into %q: %w", table, err)
	}
	return execResult(result), nil
}

func updateRows(ctx context.Context, exec sqlExecutor, table string, data map[string]any, filters []Filter) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("sqlite: update rows: no data provided")
	}
	if len(filters) == 0 {
		return 0, fmt.Errorf("sqlite: update rows: at least one filter is required")
	}
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return 0, err
	}

	columns := sortedKeys(data)
	setClauses := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns)+len(filters))
	for _, column := range columns {
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return 0, err
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoted))
		args = append(args, data[column])
	}

	whereSQL, whereArgs, err := buildWhereClause(filters)
	if err != nil {
		return 0, err
	}
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableSQL, strings.Join(setClauses, ", "), whereSQL)
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("sqlite: update rows in %q: %w", table, err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

func upsertRow(ctx context.Context, exec sqlExecutor, table string, data map[string]any, conflictColumns []string) (ExecResult, error) {
	if len(data) == 0 {
		return ExecResult{}, fmt.Errorf("sqlite: upsert row: no data provided")
	}
	if len(conflictColumns) == 0 {
		return ExecResult{}, fmt.Errorf("sqlite: upsert row: conflict columns are required")
	}

	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return ExecResult{}, err
	}

	conflicts := make(map[string]struct{}, len(conflictColumns))
	conflictSQL := make([]string, 0, len(conflictColumns))
	for _, column := range conflictColumns {
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return ExecResult{}, err
		}
		if _, ok := data[column]; !ok {
			return ExecResult{}, fmt.Errorf("sqlite: conflict column %q is missing from upsert data", column)
		}
		conflicts[column] = struct{}{}
		conflictSQL = append(conflictSQL, quoted)
	}

	columns := sortedKeys(data)
	columnSQL := make([]string, 0, len(columns))
	placeholders := make([]string, 0, len(columns))
	updateClauses := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return ExecResult{}, err
		}
		columnSQL = append(columnSQL, quoted)
		placeholders = append(placeholders, "?")
		args = append(args, data[column])
		if _, ok := conflicts[column]; !ok {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = excluded.%s", quoted, quoted))
		}
	}
	if len(updateClauses) == 0 {
		updateClauses = append(updateClauses, fmt.Sprintf("%s = %s", conflictSQL[0], conflictSQL[0]))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		tableSQL,
		strings.Join(columnSQL, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(conflictSQL, ", "),
		strings.Join(updateClauses, ", "),
	)
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return ExecResult{}, fmt.Errorf("sqlite: upsert row into %q: %w", table, err)
	}
	return execResult(result), nil
}

func deleteRows(ctx context.Context, exec sqlExecutor, table string, filters []Filter) (int64, error) {
	if len(filters) == 0 {
		return 0, fmt.Errorf("sqlite: delete rows: at least one filter is required")
	}
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return 0, err
	}
	whereSQL, args, err := buildWhereClause(filters)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableSQL, whereSQL)
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("sqlite: delete rows from %q: %w", table, err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

func execResult(result sql.Result) ExecResult {
	lastID, _ := result.LastInsertId()
	affected, _ := result.RowsAffected()
	return ExecResult{
		LastInsertID: lastID,
		RowsAffected: affected,
	}
}
