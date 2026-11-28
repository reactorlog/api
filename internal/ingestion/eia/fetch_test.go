package eia

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFetchURL(t *testing.T) {
	body := miniZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer server.Close()

	plants, generators, err := fetchURL(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(plants) != 1 || len(generators["1"]) != 1 || generators["1"][0].ID != "1" {
		t.Fatalf("plants=%+v generators=%+v", plants, generators)
	}
}

func TestFetchURLHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	if _, _, err := fetchURL(context.Background(), server.URL); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestReadBodyTooLarge(t *testing.T) {
	if _, err := readBody(bytes.NewReader([]byte("abcd")), 2); err == nil {
		t.Fatal("expected size error")
	}
}

func miniZip(t *testing.T) []byte {
	t.Helper()
	plantBook := excelize.NewFile()
	_ = plantBook.SetSheetName(plantBook.GetSheetName(0), "Plant")
	_ = plantBook.SetSheetRow("Plant", "A1", &[]any{"title"})
	_ = plantBook.SetSheetRow("Plant", "A2", &[]any{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"})
	_ = plantBook.SetSheetRow("Plant", "A3", &[]any{"1", "A", "1", "Site", "Addr", "City", "TX", "75001", "C", "32", "-96"})
	plantBytes, err := plantBook.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	genBook := excelize.NewFile()
	_ = genBook.SetSheetName(genBook.GetSheetName(0), "Operable")
	_, _ = genBook.NewSheet("Proposed")
	_, _ = genBook.NewSheet("Retired and Canceled")
	_ = genBook.SetSheetRow("Operable", "A1", &[]any{"title"})
	_ = genBook.SetSheetRow("Operable", "A2", &[]any{"x", "y", "Plant Code", "Plant Name", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"})
	_ = genBook.SetSheetRow("Operable", "A3", &[]any{"", "", "1", "Site", "1", "NUC", "100"})
	genBytes, err := genBook.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	pw, _ := zw.Create("2___Plant_Y2025.xlsx")
	_, _ = pw.Write(plantBytes.Bytes())
	gw, _ := zw.Create("3_1_Generator_Y2025.xlsx")
	_, _ = gw.Write(genBytes.Bytes())
	_ = zw.Close()
	return buf.Bytes()
}
