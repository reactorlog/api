package migrations

import "time"

type Result struct {
	Version  int
	Name     string
	Applied  bool
	Duration time.Duration
}

type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

var All = []Migration{
	{
		Version: 1,
		Name:    "create_sites_and_reactors",
		Up:      createSitesAndReactorsUp,
		Down:    createSitesAndReactorsDown,
	},
}
