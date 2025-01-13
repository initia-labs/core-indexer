package db

import "github.com/jackc/pgx/v5"

type ValidTable interface {
	Unmarshal(pgx.Rows) (map[string]interface{}, error)
}

var ValidTablesMap = map[string]ValidTable{
	"transaction_events":    &TransactionEvent{},
	"finalize_block_events": &FinalizeBlockEvent{},
	"move_events":           &MoveEvent{},
}

func isValidTableName(tableName string) bool {
	for validTable, _ := range ValidTablesMap {
		if tableName == validTable {
			return true
		}
	}
	return false
}

func GetValidTableNames() []string {
	var keys []string
	for key := range ValidTablesMap {
		keys = append(keys, key)
	}
	return keys
}
