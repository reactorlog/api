// Package eia loads EIA-860 nuclear plant metadata and matches it to site names.
package eia

// Plant is one nuclear plant from Form EIA-860.
type Plant struct {
	Code         string
	Name         string
	MatchKey     string
	AddressLine1 string
	City         string
	State        string
	Zip          string
	Latitude     float64
	Longitude    float64
}

// MatchMethod records how a site joined a plant.
type MatchMethod string

const (
	MatchExact MatchMethod = "exact"
	MatchSoft  MatchMethod = "soft"
	MatchAlias MatchMethod = "alias"
)

// MatchedPlant pairs a stored site name with its EIA plant.
type MatchedPlant struct {
	SiteName string
	Plant    Plant
	Method   MatchMethod
}

// MatchResult is the association outcome used by Import.
type MatchResult struct {
	Matched         []MatchedPlant
	UnmatchedSites  []string
	UnmatchedPlants []Plant
}
