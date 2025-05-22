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

func GetColumnsFromValidTable(table ValidTable) []string {
	switch table.(type) {
	case *TransactionEvent:
		return getColumns[TransactionEvent]()
	case *FinalizeBlockEvent:
		return getColumns[FinalizeBlockEvent]()
	case *MoveEvent:
		return getColumns[MoveEvent]()
	default:
		panic(fmt.Sprintf("unsupported table type: %T", table))
	}
}
func GetValidTableNames() []string {
	var keys []string
	for key := range ValidTablesMap {
		keys = append(keys, key)
	}
	return keys
}
