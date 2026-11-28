package eia

// siteAliases maps an NRC site name to an EIA plant code when soft matching
// is not enough. Keys are exact site.Name values from the NRC import.
var siteAliases = map[string]string{
	"Shearon Harris": "6015", // EIA name is "Harris (NC)"
}
