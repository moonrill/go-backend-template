package migrations

import "embed"

// FS contains the versioned SQL migration files (*.sql).
//
//go:embed *.sql
var FS embed.FS
