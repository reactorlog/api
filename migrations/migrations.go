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
	{Version: 1, Name: "create_sites", Up: createSitesUp, Down: createSitesDown},
	{Version: 2, Name: "create_reactors", Up: createReactorsUp, Down: createReactorsDown},
	{Version: 3, Name: "nrc_reference_data", Up: nrcReferenceDataUp, Down: nrcReferenceDataDown},
}
