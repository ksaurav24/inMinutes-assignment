package migrations

import "embed"

// Files contains the database schema migrations used by the migrate command.
//
//go:embed *.sql
var Files embed.FS
