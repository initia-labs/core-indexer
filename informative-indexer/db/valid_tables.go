package db

var validTableNames = []string{
	"transaction_events",
	"finalize_block_events",
	"move_events",
}

func isValidTableName(tableName string) bool {
	for _, validTable := range validTableNames {
		if tableName == validTable {
			return true
		}
	}
	return false
}

func GetValidTableNames() []string {
	return validTableNames
}
