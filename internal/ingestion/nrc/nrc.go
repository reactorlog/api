// Package nrc imports the NRC operating power reactor reference list.
package nrc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const ListURL = "https://www.nrc.gov/facilities-safety/operating-reactors/list-power-reactor-units"
const maxPageSize = 4 << 20

type Reactor struct {
	Name, UnitNumber, DocketNumber, ReactorType string
	SiteName, Location, State                   string
}

var docketPattern = regexp.MustCompile(`\b0(?:50|52)\d{5}\b`)
var unitPattern = regexp.MustCompile(`^(.*?)\s+(\d+)$`)
var statePattern = regexp.MustCompile(`,\s*([A-Z]{2})$`)

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func descendants(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			result = append(result, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

// Parse validates every row before returning any records. Unrecognized table
// headers and malformed rows are rejected rather than silently skipped.
func Parse(r io.Reader) ([]Reactor, error) {
	page, err := readPage(r)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(string(page)))
	if err != nil {
		return nil, fmt.Errorf("parse NRC HTML: %w", err)
	}
	table, err := reactorTable(doc)
	if err != nil {
		return nil, err
	}
	return reactorsFromTable(table)
}

func readPage(r io.Reader) ([]byte, error) {
	page, err := io.ReadAll(io.LimitReader(r, maxPageSize+1))
	if err != nil {
		return nil, fmt.Errorf("read NRC list: %w", err)
	}
	if len(page) > maxPageSize {
		return nil, fmt.Errorf("NRC list exceeds %d bytes", maxPageSize)
	}
	return page, nil
}

func isReactorTable(candidate *html.Node) bool {
	headers := descendants(candidate, "th")
	return len(headers) == 6 &&
		text(headers[0]) == "Plant Name Docket Number" &&
		text(headers[2]) == "Reactor Type" &&
		text(headers[3]) == "Location"
}

func reactorTable(doc *html.Node) (*html.Node, error) {
	var table *html.Node
	for _, candidate := range descendants(doc, "table") {
		if !isReactorTable(candidate) {
			continue
		}
		if table != nil {
			return nil, fmt.Errorf("multiple NRC reactor tables")
		}
		table = candidate
	}
	if table == nil {
		return nil, fmt.Errorf("NRC reactor table not found")
	}
	return table, nil
}

func reactorsFromTable(table *html.Node) ([]Reactor, error) {
	var reactors []Reactor
	seen := map[string]bool{}
	for _, row := range descendants(table, "tr") {
		reactor, ok, err := reactorFromRow(row, seen)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		reactors = append(reactors, reactor)
	}
	if len(reactors) == 0 {
		return nil, fmt.Errorf("NRC reactor table is empty")
	}
	return reactors, nil
}

func reactorFromRow(row *html.Node, seen map[string]bool) (Reactor, bool, error) {
	cells := descendants(row, "td")
	if len(cells) == 0 {
		return Reactor{}, false, nil
	}
	if len(cells) != 6 {
		return Reactor{}, false, fmt.Errorf("NRC row has %d columns, expected 6", len(cells))
	}
	return reactorFromCells(cells, seen)
}

func reactorFromCells(cells []*html.Node, seen map[string]bool) (Reactor, bool, error) {
	links := descendants(cells[0], "a")
	if len(links) != 1 {
		return Reactor{}, false, fmt.Errorf("NRC row is missing a unique reactor link")
	}
	record, err := reactorRecord(text(links[0]), text(cells[0]), text(cells[3]), text(cells[2]))
	if err != nil {
		return Reactor{}, false, err
	}
	if seen[record.DocketNumber] {
		return Reactor{}, false, fmt.Errorf("duplicate NRC docket %s", record.DocketNumber)
	}
	seen[record.DocketNumber] = true
	return record, true, nil
}

func reactorRecord(name, cell, location, kind string) (Reactor, error) {
	dockets := docketPattern.FindAllString(cell, -1)
	state := statePattern.FindStringSubmatch(location)
	if err := validateReactorFields(name, dockets, state, kind); err != nil {
		return Reactor{}, err
	}
	site, unit := siteAndUnit(name)
	return Reactor{
		Name: name, UnitNumber: unit, DocketNumber: dockets[0], ReactorType: kind,
		SiteName: site, Location: location, State: state[1],
	}, nil
}

func validateReactorFields(name string, dockets, state []string, kind string) error {
	if name == "" || len(dockets) != 1 || len(state) != 2 || !knownReactorType(kind) {
		return fmt.Errorf("invalid NRC reactor row %q", name)
	}
	return nil
}

func knownReactorType(kind string) bool {
	return kind == "PWR" || kind == "BWR"
}

func siteAndUnit(name string) (string, string) {
	parts := unitPattern.FindStringSubmatch(name)
	if len(parts) == 3 {
		return parts[1], parts[2]
	}
	return name, "1"
}

// Fetch uses a bounded request and rejects HTTP errors before parsing.
func Fetch(ctx context.Context) ([]Reactor, error) {
	return fetchURL(ctx, ListURL)
}

func fetchURL(ctx context.Context, pageURL string) ([]Reactor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Reactorlog/1.0 (NRC reference data ingestion)")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch NRC list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch NRC list: HTTP %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}
