package report

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	date, err := parseDate("2026-11-09", "date_from")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}
	if want := time.Date(2026, time.November, 9, 0, 0, 0, 0, time.UTC); !date.Equal(want) {
		t.Errorf("parseDate() = %v, want %v", date, want)
	}

	if _, err := parseDate("09-11-2026", "date_to"); err == nil {
		t.Fatal("parseDate() error = nil, want invalid format error")
	}
}
