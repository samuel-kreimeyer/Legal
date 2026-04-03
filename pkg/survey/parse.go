package survey

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseSurveyFile reads survey points from r. Two text formats are supported:
//
// CSV with a header row:
//
//	PT,CODE,NORTHING,EASTING,ELEVATION,DESCRIPTION
//	200,IP,568234.567,1234567.890,456.789,FOUND IRON PIN
//
// Column names are case-insensitive. Recognised aliases:
//   - point number: PT, POINT, NUMBER, NO, NAME, ID
//   - feature code: CODE, FEATURE, FEAT, FC
//   - northing:     NORTHING, NORTH, N, Y
//   - easting:      EASTING, EAST, E, X
//   - elevation:    ELEVATION, ELEV, Z, HEIGHT, HT
//   - description:  DESCRIPTION, DESC, NOTE, NOTES
//
// Space-delimited without a header:
//
//	200 IP 568234.567 1234567.890 456.789 FOUND IRON PIN
//	200 568234.567   1234567.890 456.789 IP   FOUND IRON PIN
//
// Column order is auto-detected: if the second field parses as a float it is
// treated as a northing (PT N E [Z] [CODE] [DESC…]); otherwise it is the
// feature code (PT CODE N E [Z] [DESC…]).
//
// Lines beginning with '#' and blank lines are always ignored.
func ParseSurveyFile(r io.Reader) ([]Point, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, trimmed)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	if len(lines) == 0 {
		return nil, nil
	}

	if strings.Contains(lines[0], ",") {
		return parseCSV(lines)
	}
	return parseSpaceDelimited(lines)
}

// parseCSV parses CSV lines where the first line is a header.
func parseCSV(lines []string) ([]Point, error) {
	r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV parse error: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("survey CSV has no data rows")
	}

	colPT, colCode, colN, colE, colZ, colDesc := -1, -1, -1, -1, -1, -1
	for i, h := range records[0] {
		switch strings.ToUpper(strings.TrimSpace(h)) {
		case "PT", "POINT", "NUMBER", "NO", "NAME", "ID":
			colPT = i
		case "CODE", "FEATURE", "FEAT", "FC":
			colCode = i
		case "NORTHING", "NORTH", "N", "Y":
			colN = i
		case "EASTING", "EAST", "E", "X":
			colE = i
		case "ELEVATION", "ELEV", "Z", "HEIGHT", "HT":
			colZ = i
		case "DESCRIPTION", "DESC", "NOTE", "NOTES":
			colDesc = i
		}
	}
	if colPT < 0 {
		return nil, fmt.Errorf("survey CSV missing point-number column (PT, NUMBER, NAME, etc.)")
	}
	if colCode < 0 {
		return nil, fmt.Errorf("survey CSV missing feature-code column (CODE, FEATURE, etc.)")
	}
	if colN < 0 || colE < 0 {
		return nil, fmt.Errorf("survey CSV missing coordinate columns (NORTHING/EASTING or N/E)")
	}

	pts := make([]Point, 0, len(records)-1)
	for rowIdx, row := range records[1:] {
		required := []int{colPT, colCode, colN, colE}
		for _, c := range required {
			if c >= len(row) {
				return nil, fmt.Errorf("survey CSV row %d: too few columns", rowIdx+2)
			}
		}
		pt, err := parseCSVRow(row, colPT, colCode, colN, colE, colZ, colDesc)
		if err != nil {
			return nil, fmt.Errorf("survey CSV row %d: %w", rowIdx+2, err)
		}
		pts = append(pts, pt)
	}
	return pts, nil
}

func parseCSVRow(row []string, colPT, colCode, colN, colE, colZ, colDesc int) (Point, error) {
	num, err := strconv.Atoi(strings.TrimSpace(row[colPT]))
	if err != nil {
		// Some software exports alphanumeric point names; hash them to an int.
		return Point{}, fmt.Errorf("invalid point number %q (must be an integer)", row[colPT])
	}
	north, err := strconv.ParseFloat(strings.TrimSpace(row[colN]), 64)
	if err != nil {
		return Point{}, fmt.Errorf("invalid northing %q", row[colN])
	}
	east, err := strconv.ParseFloat(strings.TrimSpace(row[colE]), 64)
	if err != nil {
		return Point{}, fmt.Errorf("invalid easting %q", row[colE])
	}
	var elev float64
	if colZ >= 0 && colZ < len(row) {
		elev, _ = strconv.ParseFloat(strings.TrimSpace(row[colZ]), 64)
	}
	var desc string
	if colDesc >= 0 && colDesc < len(row) {
		desc = strings.TrimSpace(row[colDesc])
	}
	return Point{
		Number:      num,
		Code:        FeatureCode(strings.ToUpper(strings.TrimSpace(row[colCode]))),
		Northing:    north,
		Easting:     east,
		Elevation:   elev,
		Description: desc,
	}, nil
}

// parseSpaceDelimited handles whitespace-separated survey point files without a header.
func parseSpaceDelimited(lines []string) ([]Point, error) {
	pts := make([]Point, 0, len(lines))
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("survey file line %d: expected at least 3 fields, got %d", i+1, len(fields))
		}
		num, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("survey file line %d: invalid point number %q", i+1, fields[0])
		}

		var pt Point
		pt.Number = num

		// Detect column order based on whether fields[1] is numeric.
		if _, fErr := strconv.ParseFloat(fields[1], 64); fErr == nil {
			// Order: PT  N  E  [Z]  [CODE]  [DESC...]
			pt.Northing, err = strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("survey file line %d: invalid northing %q", i+1, fields[1])
			}
			if len(fields) < 3 {
				return nil, fmt.Errorf("survey file line %d: missing easting", i+1)
			}
			pt.Easting, err = strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("survey file line %d: invalid easting %q", i+1, fields[2])
			}
			next := 3
			if next < len(fields) {
				if z, zErr := strconv.ParseFloat(fields[next], 64); zErr == nil {
					pt.Elevation = z
					next++
				}
			}
			if next < len(fields) {
				pt.Code = FeatureCode(strings.ToUpper(fields[next]))
				next++
			}
			if next < len(fields) {
				pt.Description = strings.Join(fields[next:], " ")
			}
		} else {
			// Order: PT  CODE  N  E  [Z]  [DESC...]
			pt.Code = FeatureCode(strings.ToUpper(fields[1]))
			if len(fields) < 4 {
				return nil, fmt.Errorf("survey file line %d: missing northing or easting", i+1)
			}
			pt.Northing, err = strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("survey file line %d: invalid northing %q", i+1, fields[2])
			}
			pt.Easting, err = strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, fmt.Errorf("survey file line %d: invalid easting %q", i+1, fields[3])
			}
			next := 4
			if next < len(fields) {
				if z, zErr := strconv.ParseFloat(fields[next], 64); zErr == nil {
					pt.Elevation = z
					next++
				}
			}
			if next < len(fields) {
				pt.Description = strings.Join(fields[next:], " ")
			}
		}
		pts = append(pts, pt)
	}
	return pts, nil
}
