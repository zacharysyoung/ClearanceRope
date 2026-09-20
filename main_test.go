package main

import (
	"os"
	"slices"
	"testing"

	"golang.org/x/net/html"
)

func TestParseText(t *testing.T) {
	b, err := os.ReadFile("testdata/clipboard.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseText(string(b))
	if err != nil {
		t.Fatal(err)
	}

	want := []rope{
		{"CLXR26-402", "Tree Guard, 5% Stretch", 9, 14, 6.30},
		{"CLXR26-457", "Samson Pro-Master", 21, 12, 8.38},
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
					name:       "Samson Stable Braid SamsonDry",
					lenFeet:    99,
					diameterMM: 12,
					price:      99.00,
					sku:        "CLXR26-480",
				},
				{
					name:       "Yale Prism",
					lenFeet:    63,
					diameterMM: 11.7,
					price:      64.01,
					sku:        "CLXR26-478",
				},
			},
			true,
		},
		{
			"last",
			"testdata/last.html",
			[]rope{
				{
					name:       "Arbor Plex",
					lenFeet:    47,
					diameterMM: 16,
					price:      36.65,
					sku:        "CLXR-181",
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
			ropes, more := parseProducts(doc)

			if more != tc.more {
				t.Errorf("got more=%v; want %v", more, tc.more)
			}

			if !slices.Equal(ropes, tc.ropes) {
				t.Errorf("got ropes=%v; want %v", ropes, tc.ropes)
			}
		})
	}
}

func TestParseProductName(t *testing.T) {
	testCases := []struct {
		productName string

		name       string
		lenFt      int
		diameterMM float64
	}{
		{
			"Clearance Rope: 2' Foo Bar",
			"Foo Bar",
			2,
			0,
		},
		{
			`20' Strong Stuff 11mm`,
			"Strong Stuff",
			20,
			11,
		},
		// 2" ignored because explicit MM
		{
			`30' Hang in there 12mm (2")`,
			"Hang in there",
			30,
			12,
		},
		// 0mm because 2" not listed in conversion
		{
			`40' Never gonna break 2"`,
			"Never gonna break",
			40,
			0,
		},
		// 5/8" found in conversion
		{
			`10' Rope Master 5/8"`,
			"Rope Master",
			10,
			16,
		},
		// Hyphenated diameter regexp
		{
			`9' Tree Guard - 18mm, 5% Stretch`,
			"Tree Guard, 5% Stretch",
			9,
			18,
		},
		// Diameter lookup by whole name
		{
			`78' True Blue`,
			"True Blue",
			78,
			12,
		},
		// Diameter lookup by part of name
		{
			`20' Vortex Hot`,
			"Vortex Hot",
			20,
			12.7,
		},
	}

	for _, tc := range testCases {
		name, lenFt, diameterMM := parseProductName(tc.productName)
		if name != tc.name || lenFt != tc.lenFt || diameterMM != tc.diameterMM {
			t.Errorf("for %s got name=%s, lenFt=%d, diameterMM=%g; want name=%s, lenFt=%d, diameterMM=%g",
				tc.productName,
				name, lenFt, diameterMM,
				tc.name, tc.lenFt, tc.diameterMM,
			)
		}
	}

}
