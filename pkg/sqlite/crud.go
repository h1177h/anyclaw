package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type Filter struct {
	Column string
	Op     string
	Value  any
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type sqlQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func buildWhereClause(filters []Filter) (string, []any, error) {
	clauses := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	for _, filter := range filters {
		column, err := quoteIdentifier(filter.Column)
		if err != nil {
			return "", nil, err
		}
		op, err := normalizeFilterOp(filter.Op)
		if err != nil {
			return "", nil, err
		}
		if filter.Value == nil {
			switch op {
			case "=":
				clauses = append(clauses, column+" IS NULL")
			case "!=", "<>":
				clauses = append(clauses, column+" IS NOT NULL")
			default:
				return "", nil, fmt.Errorf("sqlite: nil filter value only supports equality operators")
			}
			continue
		}
		clauses = append(clauses, column+" "+op+" ?")
		args = append(args, filter.Value)
	}
	return strings.Join(clauses, " AND "), args, nil
}

func normalizeFilterOp(op string) (string, error) {
	op = strings.ToUpper(strings.TrimSpace(op))
	if op == "" {
		op = "="
	}
	switch op {
	case "=", "!=", "<>", "<", "<=", ">", ">=", "LIKE":
		return op, nil
	default:
		return "", fmt.Errorf("sqlite: unsupported filter operator %q", op)
	}
}

func quoteQualifiedIdentifier(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("sqlite: invalid identifier %q", name)
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		q, err := quoteIdentifier(part)
		if err != nil {
			return "", err
		}
		quoted = append(quoted, q)
	}
	return strings.Join(quoted, "."), nil
}

func quoteIdentifier(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("sqlite: identifier is required")
	}
	for i, r := range name {
		isLetter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || r == '_' || i > 0 && isDigit {
			continue
		}
		return "", fmt.Errorf("sqlite: invalid identifier %q", name)
	}
	return `"` + name + `"`, nil
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scanRowsToMaps(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = values[i]
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
