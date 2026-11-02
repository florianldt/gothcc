// Package migrations embeds the goose SQL migrations, so the runtime image
// needs nothing but the binary. They are also sqlc's schema (sqlc.yaml), so a
// new column means a new migration file here: 0000N_what_changed.sql with a
// "-- +goose Up" section.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
