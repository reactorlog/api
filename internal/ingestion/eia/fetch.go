package eia

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ZipURL is the current EIA-860 annual archive. Update when EIA publishes a new year.
const ZipURL = "https://www.eia.gov/electricity/data/eia860/xls/eia8602025.zip"

const maxZipSize = 64 << 20

// Fetch downloads nuclear plants from the pinned EIA-860 zip.
func Fetch(ctx context.Context) ([]Plant, error) {
	return fetchURL(ctx, ZipURL)
}

func fetchURL(ctx context.Context, zipURL string) ([]Plant, error) {
	body, err := downloadZip(ctx, zipURL)
	if err != nil {
		return nil, err
	}
	return PlantsFromZip(body)
}

func downloadZip(ctx context.Context, zipURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Reactorlog/1.0 (EIA-860 plant ingestion)")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch EIA-860 zip: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch EIA-860 zip: HTTP %d", resp.StatusCode)
	}
	return readBody(resp.Body, maxZipSize)
}

func readBody(r io.Reader, max int) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, int64(max)+1))
	if err != nil {
		return nil, fmt.Errorf("read EIA-860 zip: %w", err)
	}
	if len(body) > max {
		return nil, fmt.Errorf("EIA-860 zip exceeds %d bytes", max)
	}
	return body, nil
}
