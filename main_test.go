package main

import (
	"slices"
	"strings"
	"testing"
)

const in = `
Sort By:
Price: Ascending

Products Per Page:
100

Columns: 1 2 3 4 6
Clearance Rope: 9' Tree Guard - 14mm, 5% Stretch
Clearance Rope: 9' Tree Guard - 14mm, 5% Stretch
Our Price $6.30
CLXR26-402
Clearance Rope: 21' Samson Pro-Master 12mm (1/2")
Clearance Rope: 21' Samson Pro-Master 12mm (1/2")
Our Price $8.38
CLXR26-457
New Clearance Ropes Added Weekly!
Check back often for new deals on clearance rope. We only ever have 1 of each length, and it's first-come, first-served! Clearance rope is non-returnable.
`

func TestParse(t *testing.T) {
	got, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	want := []rope{
		{"CLXR26-402", "Tree Guard - 14mm, 5% Stretch", 9, 6.30},
		{"CLXR26-457", "Samson Pro-Master 12mm (1/2\")", 21, 8.38},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v; want %v", got, want)
	}
}
