package main

import (
	"bufio"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	drillPositions = map[string]bool{}
	parseContour   = false
	points         []coordinates
)

func parseInput(file string) {
	input, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	scanner := bufio.NewScanner(input)

	// scan polygon apertures
	for scanner.Scan() {
		line := scanner.Text()
		parseInputLine(line)
	}

	sort.Sort(BySize(apertures))
}

func parseInputLine(line string) {
	if parseContour {
		switch {
		case strings.HasPrefix(line, "X"):
			point := parseCoordinates(line)
			points = append(points, point)
		case strings.HasPrefix(line, "G37*"):
			pos := getCenter(points)
			size := getSize(points)

			tag := pos.String()
			drillPositions[tag] = true

			a := aperture{size: size, pos: []coordinates{pos}}
			apertures = append(apertures, a)

			parseContour = false
		}
		return
	}

	switch {
	// start of file
	case strings.HasPrefix(line, "G04 Gerber Fmt "):
		parseFormat(line)

	// aperture definition
	case strings.HasPrefix(line, "%AD"):
		a := parseAperture(line)
		apertures = append(apertures, a)

	// existing aperture
	case strings.HasPrefix(line, "D"):
		name := strings.TrimSuffix(line, "*")
		currentAperture = apertureByName(name)

	// aperture coordinates
	case strings.HasPrefix(line, "X"):
		pos := parseCoordinates(line)

		tag := pos.String()
		if drillPositions[tag] {
			// log.Println("ignoring aperture", currentAperture.name)
			break
		}
		drillPositions[tag] = true

		currentAperture.pos = append(currentAperture.pos, pos)

	// start of freeform aperture
	case strings.HasPrefix(line, "G36*"):
		points = []coordinates{}
		parseContour = true
	}
}

func parseFormat(line string) {
	line = strings.TrimPrefix(line, "G04 Gerber Fmt ")
	if strings.HasSuffix(line, ", Leading zero omitted, Abs format (unit mm)*") {
		units = "metric"
		line = strings.TrimSuffix(line, ", Leading zero omitted, Abs format (unit mm)*")
	}

	format := strings.Split(line, ".")
	i, err := strconv.Atoi(format[1])
	if err != nil {
		log.Fatal("incorrect number format")
	}

	decimalDivider = math.Pow10(i)
}

func apertureByName(name string) *aperture {
	for i, e := range apertures {
		if e.name == name {
			return &apertures[i]
		}
	}
	log.Fatal("aperture not found")
	return nil
}

func parseAperture(line string) aperture {
	parts := strings.Split(line, ",")
	// get name and shape
	name := strings.TrimPrefix(parts[0], "%AD")
	shape := name[len(name)-1]
	name = name[:len(name)-1]

	// get size
	dimensions := strings.Split(strings.TrimSuffix(parts[1], "*%"), "X")
	size := 0.0

	switch shape {
	case 'C':
		d, err := strconv.ParseFloat(dimensions[0], 64)
		if err != nil {
			log.Fatal(err, line)
		}
		size = d

	case 'O':
		fallthrough
	case 'R':
		x, err := strconv.ParseFloat(dimensions[0], 64)
		if err != nil {
			log.Fatal(err, line)
		}

		y, err := strconv.ParseFloat(dimensions[1], 64)
		if err != nil {
			log.Fatal(err, line)
		}

		// best size for area
		areaSize := math.Sqrt(x * y)
		// max size for smaller dimension
		maxSize := math.Min(x, y) * 1.2

		size = math.Min(maxSize, areaSize)

	default:
		log.Fatal("unknown aperture shape: ", line)
	}

	if units == "imperial" {
		size *= 25.4
	}

	return aperture{name: name, size: size}
}

func parseCoordinates(line string) coordinates {
	line = strings.Split(line, "D")[0]
	line = strings.TrimPrefix(line, "X")
	pos := strings.Split(line, "Y")

	x, err := strconv.ParseFloat(pos[0], 64)
	if err != nil {
		log.Fatal(err, line)
	}

	y, err := strconv.ParseFloat(pos[1], 64)
	if err != nil {
		log.Fatal(err, line)
	}

	if units == "imperial" {
		x *= 25.4
		y *= 25.4
	}

	if bottom {
		x = -x
	}

	return coordinates{x: x / decimalDivider, y: y / decimalDivider}
}
