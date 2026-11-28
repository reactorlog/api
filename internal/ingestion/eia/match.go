package eia

import (
	"fmt"
	"sort"
)

// Match associates site names to plants.
// Order: exact MatchKey, siteAliases by plant code, then soft tokens.
// Duplicate exact keys on either side are an error.
// Ambiguous soft matches are an error.
// Unmatched sites are returned in the result; callers decide whether to fail.
func Match(siteNames []string, plants []Plant) (MatchResult, error) {
	plantsByExact, plantsByCode, err := indexPlants(plants)
	if err != nil {
		return MatchResult{}, err
	}
	used := map[string]bool{}
	var result MatchResult
	for _, site := range siteNames {
		matched, ok, err := matchSite(site, plants, plantsByExact, plantsByCode, used)
		if err != nil {
			return MatchResult{}, err
		}
		if !ok {
			result.UnmatchedSites = append(result.UnmatchedSites, site)
			continue
		}
		result.Matched = append(result.Matched, matched)
	}
	result.UnmatchedPlants = unusedPlants(plants, used)
	sortMatchResult(&result)
	return result, nil
}

func matchSite(
	site string,
	plants []Plant,
	byExact, byCode map[string]Plant,
	used map[string]bool,
) (MatchedPlant, bool, error) {
	exact := NormalizeName(site)
	if exact == "" {
		return MatchedPlant{}, false, fmt.Errorf("site name %q normalizes to empty", site)
	}
	if plant, ok := byExact[exact]; ok {
		return claimPlant(site, plant, MatchExact, used)
	}
	if code, ok := siteAliases[site]; ok {
		plant, ok := byCode[code]
		if !ok {
			return MatchedPlant{}, false, fmt.Errorf("alias for site %q points at missing plant code %s", site, code)
		}
		return claimPlant(site, plant, MatchAlias, used)
	}
	return matchSoft(site, plants, used)
}

func claimPlant(site string, plant Plant, method MatchMethod, used map[string]bool) (MatchedPlant, bool, error) {
	if used[plant.Code] {
		return MatchedPlant{}, false, fmt.Errorf("plant %s already matched; site %q", plant.Code, site)
	}
	used[plant.Code] = true
	return MatchedPlant{SiteName: site, Plant: plant, Method: method}, true, nil
}

func matchSoft(site string, plants []Plant, used map[string]bool) (MatchedPlant, bool, error) {
	hits := softHitsFor(site, plants, used)
	if len(hits) > 1 {
		names := make([]string, len(hits))
		for i, plant := range hits {
			names[i] = plant.Name
		}
		sort.Strings(names)
		return MatchedPlant{}, false, fmt.Errorf("ambiguous soft match for site %q: %v", site, names)
	}
	if len(hits) == 1 {
		return claimPlant(site, hits[0], MatchSoft, used)
	}
	return MatchedPlant{}, false, nil
}

func softHitsFor(site string, plants []Plant, used map[string]bool) []Plant {
	var hits []Plant
	for _, plant := range plants {
		if used[plant.Code] {
			continue
		}
		if softMatch(site, plant.Name) {
			hits = append(hits, plant)
		}
	}
	return hits
}

func unusedPlants(plants []Plant, used map[string]bool) []Plant {
	var out []Plant
	for _, plant := range plants {
		if !used[plant.Code] {
			out = append(out, plant)
		}
	}
	return out
}

func sortMatchResult(result *MatchResult) {
	sort.Strings(result.UnmatchedSites)
	sort.Slice(result.Matched, func(i, j int) bool {
		return result.Matched[i].SiteName < result.Matched[j].SiteName
	})
	sort.Slice(result.UnmatchedPlants, func(i, j int) bool {
		return result.UnmatchedPlants[i].Name < result.UnmatchedPlants[j].Name
	})
}

func indexPlants(plants []Plant) (byExact map[string]Plant, byCode map[string]Plant, err error) {
	byExact = make(map[string]Plant, len(plants))
	byCode = make(map[string]Plant, len(plants))
	for _, plant := range plants {
		if err := indexOnePlant(plant, byExact, byCode); err != nil {
			return nil, nil, err
		}
	}
	return byExact, byCode, nil
}

func indexOnePlant(plant Plant, byExact, byCode map[string]Plant) error {
	key := plant.MatchKey
	if key == "" {
		key = NormalizeName(plant.Name)
	}
	if key == "" {
		return fmt.Errorf("plant %q normalizes to empty", plant.Name)
	}
	if prev, ok := byExact[key]; ok {
		return fmt.Errorf("duplicate plant MatchKey %q (%q and %q)", key, prev.Name, plant.Name)
	}
	if prev, ok := byCode[plant.Code]; ok {
		return fmt.Errorf("duplicate plant code %s (%q and %q)", plant.Code, prev.Name, plant.Name)
	}
	plant.MatchKey = key
	byExact[key] = plant
	byCode[plant.Code] = plant
	return nil
}
