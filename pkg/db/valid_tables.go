package db

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ValidTable interface {
	Unmarshal(pgx.Rows) (map[string]any, error)
}

var ValidTablesMap = map[string]ValidTable{
	"transaction_events":    &TransactionEvent{},
	"finalize_block_events": &FinalizeBlockEvent{},
	"move_events":           &MoveEvent{},
}

func isValidTableName(tableName string) bool {
	_, ok := ValidTablesMap[tableName]
	return ok
}

func GetColumnsFromValidTable(tableName string) ([]string, error) {
	validTable, ok := ValidTablesMap[tableName]
	if !ok {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	switch validTable.(type) {
	case *TransactionEvent:
		return getColumns[TransactionEvent](), nil
	case *FinalizeBlockEvent:
		return getColumns[FinalizeBlockEvent](), nil
	case *MoveEvent:
		return getColumns[MoveEvent](), nil
	default:
		return nil, fmt.Errorf("unsupported table type: %T", validTable)
	}
}
func GetValidTableNames() []string {
	var keys []string
	for key := range ValidTablesMap {
		keys = append(keys, key)
	}
	return keys
}
