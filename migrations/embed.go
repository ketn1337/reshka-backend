// Package migrations держит embed.FS с .sql-файлами,
// чтобы cmd/server мог мигрировать изнутри бинаря.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
