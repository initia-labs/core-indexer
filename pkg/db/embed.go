package db

import (
	"embed"
)

//go:embed custom_types.sql
var customTypesSQLFile embed.FS
