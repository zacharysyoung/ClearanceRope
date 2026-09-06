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
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	s, err := readAll()
	if err != nil {
		exit(err.Error())
	}
	fmt.Println("read from clipboard")
	out, err := Main(strings.NewReader(s))
	if err != nil {
		exit(err.Error())
	}
	err = writeAll(out)
	if err != nil {
		exit(err.Error())
	}
	fmt.Println("wrote to clipboard")
}

type rope struct {
	id, name string
	lenFeet  int
	price    float64
}

func Main(r io.Reader) (string, error) {
	ropes, err := parse(r)
	if err != nil {
		exit("%v", err)
	}

	out := &bytes.Buffer{}
	w := csv.NewWriter(out)
	w.Write([]string{"ID", "Name", "Len (ft)", "Price ($)"})
	for _, rope := range ropes {
		w.Write([]string{
			rope.id,
			rope.name,
			fmt.Sprintf("%d", rope.lenFeet),
			fmt.Sprintf("%.2f", rope.price),
		})
	}
	w.Flush()

	return out.String(), nil
}

var trim = strings.TrimSpace

func parse(r io.Reader) ([]rope, error) {
	ropes := []rope{}
	more := false

	scanner := bufio.NewScanner(r)
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

		// Product ID
		more = scanner.Scan()
		if !more {
			return nil, fmt.Errorf("scanning rope %d, line 4: no id", i)
		}
		id := trim(scanner.Text())

		ropes = append(ropes, rope{
			id:      id,
			name:    name,
			lenFeet: len,
			price:   price,
		})
	}

	return ropes, nil
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
