package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Order struct {
	Column string
	Desc   bool
}

type SelectOptions struct {
	Columns  []string
	Filters  []Filter
	Orders   []Order
	Limit    int
	Offset   int
	Distinct bool
}

func (db *DB) SelectRows(ctx context.Context, table string, opts SelectOptions) ([]map[string]any, error) {
	return selectRows(ctx, db, table, opts)
}

func (tx *Transaction) SelectRows(ctx context.Context, table string, opts SelectOptions) ([]map[string]any, error) {
	if err := tx.ensureActive(); err != nil {
		return nil, err
	}
	return selectRows(ctx, tx.tx, table, opts)
}

func (db *DB) SelectOne(ctx context.Context, table string, opts SelectOptions) (map[string]any, error) {
	return selectOne(ctx, db, table, opts)
}

func (tx *Transaction) SelectOne(ctx context.Context, table string, opts SelectOptions) (map[string]any, error) {
	if err := tx.ensureActive(); err != nil {
		return nil, err
	}
	return selectOne(ctx, tx.tx, table, opts)
}

func (db *DB) CountRows(ctx context.Context, table string, filters ...Filter) (int64, error) {
	return countRows(ctx, db, table, filters)
}

func (tx *Transaction) CountRows(ctx context.Context, table string, filters ...Filter) (int64, error) {
	if err := tx.ensureActive(); err != nil {
		return 0, err
	}
	return countRows(ctx, tx.tx, table, filters)
}

func (db *DB) RowExists(ctx context.Context, table string, filters ...Filter) (bool, error) {
	return rowExists(ctx, db, table, filters)
}

func (tx *Transaction) RowExists(ctx context.Context, table string, filters ...Filter) (bool, error) {
	if err := tx.ensureActive(); err != nil {
		return false, err
	}
	return rowExists(ctx, tx.tx, table, filters)
}

func selectRows(ctx context.Context, querier sqlQuerier, table string, opts SelectOptions) ([]map[string]any, error) {
	query, args, err := buildSelect(table, opts)
	if err != nil {
		return nil, err
	}
	rows, err := querier.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: select rows from %q: %w", table, err)
	}
	defer rows.Close()
	return scanRowsToMaps(rows)
}

func selectOne(ctx context.Context, querier sqlQuerier, table string, opts SelectOptions) (map[string]any, error) {
	opts.Limit = 1
	rows, err := selectRows(ctx, querier, table, opts)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	return rows[0], nil
}

func countRows(ctx context.Context, querier sqlQuerier, table string, filters []Filter) (int64, error) {
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableSQL)
	args := []any(nil)
	if len(filters) > 0 {
		whereSQL, whereArgs, err := buildWhereClause(filters)
		if err != nil {
			return 0, err
		}
		query += " WHERE " + whereSQL
		args = whereArgs
	}

	var count int64
	if err := querier.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("sqlite: count rows in %q: %w", table, err)
	}
	return count, nil
}

func rowExists(ctx context.Context, querier sqlQuerier, table string, filters []Filter) (bool, error) {
	query, args, err := buildExists(table, filters)
	if err != nil {
		return false, err
	}

	var marker int
	if err := querier.QueryRowContext(ctx, query, args...).Scan(&marker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("sqlite: check row exists in %q: %w", table, err)
	}
	return true, nil
}

func buildExists(table string, filters []Filter) (string, []any, error) {
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return "", nil, err
	}
	query := fmt.Sprintf("SELECT 1 FROM %s", tableSQL)
	args := []any(nil)
	if len(filters) > 0 {
		whereSQL, whereArgs, err := buildWhereClause(filters)
		if err != nil {
			return "", nil, err
		}
		query += " WHERE " + whereSQL
		args = whereArgs
	}
	query += " LIMIT 1"
	return query, args, nil
}

func buildSelect(table string, opts SelectOptions) (string, []any, error) {
	tableSQL, err := quoteQualifiedIdentifier(table)
	if err != nil {
		return "", nil, err
	}

	columnsSQL := "*"
	if len(opts.Columns) > 0 {
		columns := make([]string, 0, len(opts.Columns))
		for _, column := range opts.Columns {
			quoted, err := quoteIdentifier(column)
			if err != nil {
				return "", nil, err
			}
			columns = append(columns, quoted)
		}
		columnsSQL = strings.Join(columns, ", ")
	}

	distinct := ""
	if opts.Distinct {
		distinct = "DISTINCT "
	}
	query := fmt.Sprintf("SELECT %s%s FROM %s", distinct, columnsSQL, tableSQL)

	args := []any(nil)
	if len(opts.Filters) > 0 {
		whereSQL, whereArgs, err := buildWhereClause(opts.Filters)
		if err != nil {
			return "", nil, err
		}
		query += " WHERE " + whereSQL
		args = whereArgs
	}

	if len(opts.Orders) > 0 {
		orders := make([]string, 0, len(opts.Orders))
		for _, order := range opts.Orders {
			quoted, err := quoteIdentifier(order.Column)
			if err != nil {
				return "", nil, err
			}
			direction := "ASC"
			if order.Desc {
				direction = "DESC"
			}
			orders = append(orders, quoted+" "+direction)
		}
		query += " ORDER BY " + strings.Join(orders, ", ")
	}

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	} else if opts.Offset > 0 {
		query += " LIMIT -1"
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	return query, args, nil
}
