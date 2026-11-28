package eia

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	plantSheetName = "Plant"
	genSheetName   = "Operable"
	energyNUC      = "NUC"
)

// PlantsFromZip reads nuclear plants and operable NUC generators from an EIA-860 zip.
// Generators are indexed by plant code and are not stored on Plant.
func PlantsFromZip(zipBytes []byte) ([]Plant, map[string][]Generator, error) {
	plantName, genName, err := eia860Members(zipBytes)
	if err != nil {
		return nil, nil, err
	}
	plantRows, err := readZipSheet(zipBytes, plantName, plantSheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("read plant workbook: %w", err)
	}
	genRows, err := readZipSheet(zipBytes, genName, genSheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("read generator workbook: %w", err)
	}
	return plantsFromTables(plantRows, genRows)
}

func plantsFromTables(plantRows, genRows [][]string) ([]Plant, map[string][]Generator, error) {
	generators, err := nuclearGenerators(genRows)
	if err != nil {
		return nil, nil, err
	}
	if len(generators) == 0 {
		return nil, nil, fmt.Errorf("no operable NUC generators found")
	}
	plants, err := plantsForCodes(plantRows, generatorCodes(generators))
	if err != nil {
		return nil, nil, err
	}
	return plants, generators, nil
}

func generatorCodes(generators map[string][]Generator) map[string]bool {
	codes := make(map[string]bool, len(generators))
	for code := range generators {
		codes[code] = true
	}
	return codes
}

func plantsForCodes(plantRows [][]string, nuclearCodes map[string]bool) ([]Plant, error) {
	header, data, err := splitHeader(plantRows)
	if err != nil {
		return nil, fmt.Errorf("plant sheet: %w", err)
	}
	cols, err := plantColumns(header)
	if err != nil {
		return nil, err
	}
	plants, seen, err := collectNuclearPlants(data, cols, nuclearCodes)
	if err != nil {
		return nil, err
	}
	return finishPlants(plants, nuclearCodes, seen)
}

func finishPlants(plants []Plant, nuclearCodes, seen map[string]bool) ([]Plant, error) {
	if err := ensureCodesPresent(nuclearCodes, seen); err != nil {
		return nil, err
	}
	if len(plants) == 0 {
		return nil, fmt.Errorf("no nuclear plants parsed")
	}
	return plants, nil
}

func collectNuclearPlants(data [][]string, cols plantCols, nuclearCodes map[string]bool) ([]Plant, map[string]bool, error) {
	var plants []Plant
	seen := map[string]bool{}
	for i, row := range data {
		plant, code, ok, err := nuclearPlantFromRow(row, cols, nuclearCodes, seen, i)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		seen[code] = true
		plants = append(plants, plant)
	}
	return plants, seen, nil
}

func nuclearPlantFromRow(row []string, cols plantCols, nuclearCodes, seen map[string]bool, i int) (Plant, string, bool, error) {
	code, ok := nuclearCodeInRow(row, cols.code, nuclearCodes)
	if !ok {
		return Plant{}, "", false, nil
	}
	if seen[code] {
		return Plant{}, "", false, fmt.Errorf("duplicate plant code %s on plant sheet row %d", code, i+2)
	}
	plant, err := plantFromRow(row, cols, code)
	if err != nil {
		return Plant{}, "", false, fmt.Errorf("plant code %s: %w", code, err)
	}
	return plant, code, true, nil
}

func nuclearCodeInRow(row []string, codeIdx int, nuclearCodes map[string]bool) (string, bool) {
	code, err := cell(row, codeIdx)
	if err != nil || code == "" || strings.HasPrefix(code, "NOTE") {
		return "", false
	}
	code = normalizeCode(code)
	if !nuclearCodes[code] {
		return "", false
	}
	return code, true
}

func ensureCodesPresent(nuclearCodes, seen map[string]bool) error {
	for code := range nuclearCodes {
		if !seen[code] {
			return fmt.Errorf("nuclear plant code %s missing from plant sheet", code)
		}
	}
	return nil
}

type plantCols struct {
	code, name, address, city, state, zip, lat, lon int
}

func plantColumns(header []string) (plantCols, error) {
	cols := plantCols{}
	need := map[string]*int{
		"Plant Code":     &cols.code,
		"Plant Name":     &cols.name,
		"Street Address": &cols.address,
		"City":           &cols.city,
		"State":          &cols.state,
		"Zip":            &cols.zip,
		"Latitude":       &cols.lat,
		"Longitude":      &cols.lon,
	}
	index := headerIndex(header)
	for name, ptr := range need {
		i, ok := index[name]
		if !ok {
			return plantCols{}, fmt.Errorf("plant sheet missing column %q", name)
		}
		*ptr = i
	}
	return cols, nil
}

func headerIndex(header []string) map[string]int {
	index := map[string]int{}
	for i, name := range header {
		name = strings.TrimSpace(name)
		if name != "" {
			index[name] = i
		}
	}
	return index
}

func plantFromRow(row []string, cols plantCols, code string) (Plant, error) {
	name, err := requiredCell(row, cols.name, "Plant Name")
	if err != nil {
		return Plant{}, err
	}
	lat, lon, err := parseCoords(row, cols)
	if err != nil {
		return Plant{}, err
	}
	address, _ := cell(row, cols.address)
	city, _ := cell(row, cols.city)
	state, _ := cell(row, cols.state)
	zip, _ := cell(row, cols.zip)
	return Plant{
		Code:         code,
		Name:         name,
		MatchKey:     NormalizeName(name),
		AddressLine1: strings.TrimSpace(address),
		City:         strings.TrimSpace(city),
		State:        strings.TrimSpace(state),
		Zip:          normalizeZip(zip),
		Latitude:     lat,
		Longitude:    lon,
	}, nil
}

func parseCoords(row []string, cols plantCols) (float64, float64, error) {
	lat, err := parseFloatCell(row, cols.lat, "Latitude")
	if err != nil {
		return 0, 0, err
	}
	lon, err := parseFloatCell(row, cols.lon, "Longitude")
	if err != nil {
		return 0, 0, err
	}
	if !validCoords(lat, lon) {
		return 0, 0, fmt.Errorf("coordinates out of range (%v, %v)", lat, lon)
	}
	return lat, lon, nil
}

func validCoords(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

type genCols struct {
	code, energy, id, mw int
}

func nuclearGenerators(genRows [][]string) (map[string][]Generator, error) {
	header, data, err := splitHeader(genRows)
	if err != nil {
		return nil, fmt.Errorf("generator sheet: %w", err)
	}
	cols, err := generatorColumns(header)
	if err != nil {
		return nil, err
	}
	return collectGenerators(data, cols)
}

func generatorColumns(header []string) (genCols, error) {
	code, err := columnIndex(header, "Plant Code")
	if err != nil {
		return genCols{}, err
	}
	energy, err := columnIndex(header, "Energy Source 1")
	if err != nil {
		return genCols{}, err
	}
	id, err := columnIndex(header, "Generator ID")
	if err != nil {
		return genCols{}, err
	}
	mw, err := columnIndex(header, "Nameplate Capacity (MW)")
	if err != nil {
		return genCols{}, err
	}
	return genCols{code: code, energy: energy, id: id, mw: mw}, nil
}

func collectGenerators(data [][]string, cols genCols) (map[string][]Generator, error) {
	out := map[string][]Generator{}
	for i, row := range data {
		generator, ok, err := generatorFromRow(row, cols)
		if err != nil {
			return nil, fmt.Errorf("generator sheet row %d: %w", i+2, err)
		}
		if !ok {
			continue
		}
		out[generator.PlantCode] = append(out[generator.PlantCode], generator)
	}
	return out, nil
}

func generatorFromRow(row []string, cols genCols) (Generator, bool, error) {
	code, ok := generatorPlantCode(row, cols)
	if !ok {
		return Generator{}, false, nil
	}
	if !rowIsNUC(row, cols) {
		return Generator{}, false, nil
	}
	return nucGenerator(row, cols, code)
}

func generatorPlantCode(row []string, cols genCols) (string, bool) {
	code, err := cell(row, cols.code)
	if err != nil || code == "" || strings.HasPrefix(code, "NOTE") {
		return "", false
	}
	return normalizeCode(code), true
}

func rowIsNUC(row []string, cols genCols) bool {
	fuel, _ := cell(row, cols.energy)
	return strings.TrimSpace(fuel) == energyNUC
}

func nucGenerator(row []string, cols genCols, code string) (Generator, bool, error) {
	id, err := requiredCell(row, cols.id, "Generator ID")
	if err != nil {
		return Generator{}, false, err
	}
	mw, err := parseFloatCell(row, cols.mw, "Nameplate Capacity (MW)")
	if err != nil {
		return Generator{}, false, err
	}
	return Generator{PlantCode: code, ID: normalizeCode(id), NameplateMW: mw}, true, nil
}

func splitHeader(rows [][]string) ([]string, [][]string, error) {
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("empty sheet")
	}
	start := headerStart(rows)
	if !containsHeader(rows[start], "Plant Code") {
		return nil, nil, fmt.Errorf("header row with Plant Code not found")
	}
	return rows[start], rows[start+1:], nil
}

func headerStart(rows [][]string) int {
	if containsHeader(rows[0], "Plant Code") {
		return 0
	}
	if len(rows) > 1 && containsHeader(rows[1], "Plant Code") {
		return 1
	}
	return 0
}

func containsHeader(row []string, name string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) == name {
			return true
		}
	}
	return false
}

func columnIndex(header []string, name string) (int, error) {
	for i, cell := range header {
		if strings.TrimSpace(cell) == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("missing column %q", name)
}

func cell(row []string, i int) (string, error) {
	if i < 0 || i >= len(row) {
		return "", fmt.Errorf("column %d out of range", i)
	}
	return strings.TrimSpace(row[i]), nil
}

func requiredCell(row []string, i int, label string) (string, error) {
	value, err := cell(row, i)
	if err != nil || value == "" {
		return "", fmt.Errorf("missing %s", label)
	}
	return value, nil
}

func parseFloatCell(row []string, i int, label string) (float64, error) {
	value, err := requiredCell(row, i, label)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", label, value)
	}
	return f, nil
}

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	if f, err := strconv.ParseFloat(code, 64); err == nil {
		return strconv.FormatInt(int64(f), 10)
	}
	return code
}

func normalizeZip(zip string) string {
	zip = strings.TrimSpace(zip)
	if zip == "" {
		return ""
	}
	f, err := strconv.ParseFloat(zip, 64)
	if err != nil {
		return zip
	}
	return padZip(int64(f))
}

func padZip(n int64) string {
	if n >= 0 && n <= 99999 {
		return fmt.Sprintf("%05d", n)
	}
	return strconv.FormatInt(n, 10)
}

func readZipSheet(zipBytes []byte, member, sheet string) ([][]string, error) {
	raw, err := readZipMember(zipBytes, member)
	if err != nil {
		return nil, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetRows(sheet)
}
