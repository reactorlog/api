package eia

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestEia860MembersAndPlantsFromZip(t *testing.T) {
	plantBook := excelize.NewFile()
	plantSheet := plantBook.GetSheetName(0)
	_ = plantBook.SetSheetName(plantSheet, "Plant")
	_ = plantBook.SetSheetRow("Plant", "A1", &[]any{"title"})
	_ = plantBook.SetSheetRow("Plant", "A2", &[]any{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"})
	_ = plantBook.SetSheetRow("Plant", "A3", &[]any{"1", "A", "6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "SLO", "35.2", "-120.8"})
	plantBytes, err := plantBook.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	genBook := excelize.NewFile()
	_ = genBook.SetSheetName(genBook.GetSheetName(0), "Operable")
	_, _ = genBook.NewSheet("Proposed")
	_, _ = genBook.NewSheet("Retired and Canceled")
	_ = genBook.SetSheetRow("Operable", "A1", &[]any{"title"})
	_ = genBook.SetSheetRow("Operable", "A2", &[]any{"x", "y", "Plant Code", "Plant Name", "Energy Source 1"})
	_ = genBook.SetSheetRow("Operable", "A3", &[]any{"", "", "6099", "Diablo Canyon", "NUC"})
	genBytes, err := genBook.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	plantW, err := zw.Create("2___Plant_Y2025.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plantW.Write(plantBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	genW, err := zw.Create("3_1_Generator_Y2025.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := genW.Write(genBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	plants, err := PlantsFromZip(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(plants) != 1 || plants[0].Code != "6099" {
		t.Fatalf("%+v", plants)
	}
}

func TestEia860MembersMissing(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("readme.txt")
	_, _ = w.Write([]byte("hi"))
	_ = zw.Close()
	if _, _, err := eia860Members(buf.Bytes()); err == nil {
		t.Fatal("expected missing workbooks")
	}
}

func TestReadZipMemberMissing(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_ = zw.Close()
	if _, err := readZipMember(buf.Bytes(), "nope.xlsx"); err == nil {
		t.Fatal("expected missing member")
	}
}

func TestAssignMemberDuplicate(t *testing.T) {
	if _, _, err := assignMember("2___Plant_Y2025.xlsx", "a.xlsx", "already", ""); err == nil {
		t.Fatal("expected duplicate plant")
	}
	if _, _, err := assignMember("3_1_Generator_Y2025.xlsx", "b.xlsx", "", "already"); err == nil {
		t.Fatal("expected duplicate generator")
	}
}
