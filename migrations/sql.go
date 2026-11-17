package migrations

import _ "embed"

//go:embed sql/migrations_table.sql
var migrationsTableSQL string

//go:embed sql/applied_versions.sql
var appliedVersionsSQL string

//go:embed sql/record_migration.sql
var recordMigrationSQL string

//go:embed sql/1_create_sites_and_reactors.up.sql
var createSitesAndReactorsUp string

//go:embed sql/1_create_sites_and_reactors.down.sql
var createSitesAndReactorsDown string
