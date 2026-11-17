package migrations

import _ "embed"

//go:embed sql/migrations_table.sql
var migrationsTableSQL string

//go:embed sql/applied_versions.sql
var appliedVersionsSQL string

//go:embed sql/record_migration.sql
var recordMigrationSQL string

//go:embed sql/1_create_sites.up.sql
var createSitesUp string

//go:embed sql/1_create_sites.down.sql
var createSitesDown string

//go:embed sql/2_create_reactors.up.sql
var createReactorsUp string

//go:embed sql/2_create_reactors.down.sql
var createReactorsDown string

//go:embed sql/3_nrc_reference_data.up.sql
var nrcReferenceDataUp string

//go:embed sql/3_nrc_reference_data.down.sql
var nrcReferenceDataDown string
