package main

import (
	"os"
	"slices"
	"testing"

	"golang.org/x/net/html"
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

func TestParseText(t *testing.T) {
	got, err := parseText(in)
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

func TestParseHTML(t *testing.T) {
	testCases := []struct {
		name     string
		htmlfile string
		ropes    []rope
		more     bool
	}{
		{
			"first",
			"testdata/first.html",
			[]rope{
				{
					name:    "Samson Stable Braid SamsonDry 12mm (1/2\")",
					lenFeet: 99,
					price:   99.00,
					id:      "CLXR26-480",
				},
				{
					name:    "Yale Prism 11.7mm",
					lenFeet: 63,
					price:   64.01,
					id:      "CLXR26-478",
				},
			},
			true,
		},
		{
			"last",
			"testdata/last.html",
			[]rope{
				{
					name:    "Arbor Plex 5/8\"",
					lenFeet: 47,
					price:   36.65,
					id:      "CLXR-181",
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.htmlfile)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := html.Parse(f)
			if err != nil {
				t.Fatal(err)
			}
			ropes, more := parseHTML(doc)

			if more != tc.more {
				t.Errorf("got more=%v; want %v", more, tc.more)
			}

			if !slices.Equal(ropes, tc.ropes) {
				t.Errorf("got ropes=%v; want %v", ropes, tc.ropes)
			}
		})
	}
}
