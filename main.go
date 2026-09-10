// Parses WesSpur's Clearance Rope section and outputs a
// sortable CSV.
//
// 1. Set Products per Page to 100 (https://www.wesspur.com/specials/clearance-rope?mode=6&sort=newest&limit=1000)
// 2. Select all and copy
// 3. Run this program
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	webflag   = flag.Bool("web", false, "scrape web")
	printflag = flag.Bool("print", false, "print CSV instead of copying to clipboard")
)

func main() {
	flag.Parse()
	var (
		ropes []rope
		err   error
	)

	if *webflag {
		ropes, err = MainHTML()
	} else {
		ropes, err = MainText()
	}
	if err != nil {
		exit(err.Error())
	}

	sort.Slice(ropes, func(i, j int) bool {
		return ropes[i].lenFeet < ropes[j].lenFeet
	})

	s := toCSV(ropes)
	switch *printflag {
	case true:
		fmt.Println(s)
	default:
		err = writeAll(s)
		if err != nil {
			exit(err.Error())
		}
		fmt.Println("wrote to clipboard")
	}
}

type rope struct {
	sku, name  string
	lenFeet    int
	diameterMM float64
	price      float64
}

func MainText() ([]rope, error) {
	s, err := readAll()
	if err != nil {
		return nil, err
	}
	fmt.Println("read from clipboard")

	return parseText(s)
}

var trim = strings.TrimSpace

func parseText(s string) ([]rope, error) {
	ropes := []rope{}
	more := false

	scanner := bufio.NewScanner(strings.NewReader(s))
	for i := 1; scanner.Scan(); i++ {
		line := trim(scanner.Text())

		if !strings.HasPrefix(line, "Clearance Rope:") {
			continue
		}

		// Description, e.g., Clearance Rope: 9' Tree Guard - 14mm, 5% Stretch
		line = trim(strings.TrimPrefix(line, "Clearance Rope:"))
		parts := strings.SplitN(line, "'", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("parsing rope %d, line 2: no feet designator \"'\" in %q", i, line)
		}
		len, err := strconv.Atoi(trim(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("could not parse length: %v", err)
		}
		name := trim(parts[1])

		// Duplicate description, discard
		more = scanner.Scan()
		if !more {
			return nil, fmt.Errorf("scanning rope %d, line 2: no second rope description", i)
		}

		// Price
		more = scanner.Scan()
		if !more {
			return nil, fmt.Errorf("scanning rope %d, line 3: no price", i)
		}
		line = trim(scanner.Text())
		if !strings.HasPrefix(line, "Our Price $") {
			return nil, fmt.Errorf("parsing rope %d, line 3: no $ in %q", i, line)
		}
		line = strings.TrimPrefix(line, "Our Price $")
		price, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse price: %v", err)
		}

		// Product SKU
		more = scanner.Scan()
		if !more {
			return nil, fmt.Errorf("scanning rope %d, line 4: no SKU", i)
		}
		sku := trim(scanner.Text())

		ropes = append(ropes, rope{
			sku:     sku,
			name:    name,
			lenFeet: len,
			price:   price,
		})
	}

	return ropes, nil
}

func MainHTML() ([]rope, error) {
	page := 1
	ropes := []rope{}

	for true {
		r, err := getHTML(page)
		if err != nil {
			return nil, err
		}
		doc, err := html.Parse(r)
		if err != nil {
			return nil, err
		}
		_ropes, more := parseProducts(doc)
		fmt.Fprintf(os.Stderr, "scraped page %d, got %d ropes\n", page, len(_ropes))
		ropes = append(ropes, _ropes...)
		if !more {
			break
		}
		page++
	}
	return ropes, nil
}

func parseProducts(doc *html.Node) (ropes []rope, more bool) {
	for n := range doc.Descendants() {
		switch {
		case elementHasClass(n, atom.Ul, "productGrid"):
			for n := range n.ChildNodes() {
				if n.Type == html.ElementNode && n.DataAtom == atom.Li {
					rope := scrapeRope(n)
					ropes = append(ropes, rope)
				}
			}
		case elementHasClass(n, atom.Li, "pagination-item--next"):
			more = true
		}
	}

	return ropes, more
}

func scrapeRope(n *html.Node) rope {
	rope := rope{}
	for n := range n.Descendants() {
		if elementHasClass(n, atom.H4, "card-title") {
			s := getInnerText(n)
			name, lenFt, diameterMM := parseProductName(s)
			rope.name = name
			rope.lenFeet = lenFt
			rope.diameterMM = diameterMM
		}

		// the price--main class appears multiple times,
		// sometimes w/out actual price text
		if elementHasClass(n, atom.Span, "price--main") && rope.price == 0 {
			s := getInnerText(n)
			s = strings.TrimPrefix(s, "$")
			f, _ := strconv.ParseFloat(s, 64)
			rope.price = f
		}

		if elementHasClass(n, atom.Div, "card-text--sku") {
			rope.sku = getInnerText(n)
		}
	}
	return rope
}

// Parse a product name like `Clearance Rope: 99' Samson Stable Braid SamsonDry 12mm (1/2\")`
// into "Samson Stable Braid SamsonDry", 99, 12.0.
func parseProductName(s string) (name string, lenFt int, diameterMM float64) {
	s = strings.TrimPrefix(s, "Clearance Rope: ")
	parts := strings.Split(s, " ")

	{
		s := parts[0]
		s = strings.TrimSuffix(s, "'")
		lenFt, _ = strconv.Atoi(s)
		parts = parts[1:]
	}

	i := 0
	part := ""
	hasDiameter := false
PartsLoop:
	for i, part = range parts {
		switch {
		case strings.HasSuffix(part, "mm"),
			strings.HasSuffix(part, "("),
			strings.HasSuffix(part, `"`):
			hasDiameter = true
			break PartsLoop
		}
	}

	if !hasDiameter {
		name = strings.Join(parts, " ")
		return
	}

	name = strings.Join(parts[:i], " ")
	parts = parts[i:]

	if strings.HasSuffix(parts[0], "mm") {
		s := strings.TrimSuffix(parts[0], "mm")
		diameterMM, _ = strconv.ParseFloat(s, 64)
		if i < (len(parts) - 1) {
			parts = parts[i+1:]
		}
	}

	if diameterMM == 0 &&
		(strings.HasPrefix(parts[0], "(") || strings.HasSuffix(parts[0], `"`)) {
		conversion := map[string]float64{
			`3/8"`:  9,
			`1/2"`:  12,
			`9/16"`: 14,
			`5/8"`:  16,
			`3/4"`:  19,
			`1"`:    24,
		}

		s := strings.TrimSuffix(strings.TrimPrefix(parts[0], "("), ")")
		diameterMM = conversion[s]

		if diameterMM == 0 {
			fmt.Println("could not find conversion for ", s)
		}
	}

	return
}

func elementHasClass(n *html.Node, atom atom.Atom, className string) bool {
	if n.Type == html.ElementNode && n.DataAtom == atom {
		for _, a := range n.Attr {
			if a.Key == "class" {
				for _, s := range strings.Split(a.Val, " ") {
					if s == className {
						return true
					}
				}
			}
		}
	}
	return false
}

var allSpaces = regexp.MustCompile(`\s+`)

func getInnerText(n *html.Node) string {
	txt := ""
	for n := range n.Descendants() {
		if n.Type == html.TextNode {
			s := trim(n.Data)
			if s == "" {
				continue
			}
			txt += " " + allSpaces.ReplaceAllString(s, " ")
		}
	}
	return trim(txt)
}

func getHTML(page int) (io.ReadCloser, error) {
	url := fmt.Sprintf("https://www.wesspur.com/specials/clearance-rope?limit=100&mode=4&page=%d", page)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not get page %d: %v", page, err)
	}
	if code := resp.StatusCode; code != 200 {
		return nil, fmt.Errorf("got non-200 getting page %d: %d", page, code)
	}
	return resp.Body, nil
}

func toCSV(ropes []rope) string {
	out := &bytes.Buffer{}
	w := csv.NewWriter(out)
	w.Write([]string{"Name", "Len (ft)", "Diameter (mm)", "Price ($)", "SKU"})
	for _, rope := range ropes {
		w.Write([]string{
			rope.name,
			fmt.Sprintf("%d", rope.lenFeet),
			fmt.Sprintf("%g", rope.diameterMM),
			fmt.Sprintf("%.2f", rope.price),
			rope.sku,
		})
	}
	w.Flush()

	return out.String()
}

func exit(format string, args ...any) {
	if !strings.HasPrefix(format, "error: ") {
		format = "error: " + format
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(2)
}
