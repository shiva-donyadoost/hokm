// Package migrations embeds the SQL migration files so any binary can apply
// them at startup without external tooling.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
