package eia

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
)

func eia860Members(zipBytes []byte) (string, string, error) {
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return "", "", fmt.Errorf("open EIA-860 zip: %w", err)
	}
	return findWorkbooks(r.File)
}

func findWorkbooks(files []*zip.File) (plantName, genName string, err error) {
	for _, f := range files {
		plantName, genName, err = assignMember(path.Base(f.Name), f.Name, plantName, genName)
		if err != nil {
			return "", "", err
		}
	}
	if plantName == "" || genName == "" {
		return "", "", fmt.Errorf("EIA-860 zip missing plant or generator workbook")
	}
	return plantName, genName, nil
}

func assignMember(base, full, plantName, genName string) (string, string, error) {
	if isPlantWorkbook(base) {
		return setOnce(full, plantName, genName, true)
	}
	if isGeneratorWorkbook(base) {
		return setOnce(full, plantName, genName, false)
	}
	return plantName, genName, nil
}

func isPlantWorkbook(base string) bool {
	return strings.HasPrefix(base, "2___Plant_Y") && strings.HasSuffix(base, ".xlsx")
}

func isGeneratorWorkbook(base string) bool {
	return strings.HasPrefix(base, "3_1_Generator_Y") && strings.HasSuffix(base, ".xlsx")
}

func setOnce(full, plantName, genName string, plant bool) (string, string, error) {
	if plant {
		if plantName != "" {
			return "", "", fmt.Errorf("multiple plant workbooks in EIA-860 zip")
		}
		return full, genName, nil
	}
	if genName != "" {
		return "", "", fmt.Errorf("multiple generator workbooks in EIA-860 zip")
	}
	return plantName, full, nil
}

func readZipMember(zipBytes []byte, member string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open EIA-860 zip: %w", err)
	}
	for _, f := range r.File {
		if f.Name != member {
			continue
		}
		return readZipFile(f)
	}
	return nil, fmt.Errorf("EIA-860 zip missing %s", member)
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	const maxWorkbook = 64 << 20
	data, err := io.ReadAll(io.LimitReader(rc, maxWorkbook+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxWorkbook {
		return nil, fmt.Errorf("EIA-860 member %s exceeds %d bytes", f.Name, maxWorkbook)
	}
	return data, nil
}
