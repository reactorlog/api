package nrc

import (
	"os"
	"strings"
	"testing"
)

func TestVerificationAmbiguousTables(t *testing.T) {
	source, err := os.ReadFile("testdata/operating-units.html")
	if err != nil {
		t.Fatal(err)
	}
	records, err := Parse(strings.NewReader(string(source) + string(source)))
	if err == nil || !strings.Contains(err.Error(), "multiple NRC reactor tables") || len(records) != 0 {
		t.Fatalf("expected ambiguous-table rejection, records=%d error=%v", len(records), err)
	}
}
