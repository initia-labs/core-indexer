package db

import (
	"gorm.io/gorm/schema"
)

var ValidTablesMap = map[string]schema.Tabler{
	"transaction_events":    &TransactionEvent{},
	"finalize_block_events": &FinalizeBlockEvent{},
	"move_events":           &MoveEvent{},
}

func isValidTableName(tableName string) bool {
	_, ok := ValidTablesMap[tableName]
	return ok
}

func GetValidTableNames() []string {
	var keys []string
	for key := range ValidTablesMap {
		keys = append(keys, key)
	}
	return keys
}
