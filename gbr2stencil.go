package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	iterationCount  = 100
	outputExtension = ".Stencil.ngc"
	outputHeader    = `G94 ( Millimeters per minute feed rate. )
G21 ( Units == Millimeters. )

G90 ( Absolute coordinates. )
S10000 ( RPM spindle speed. )
G64 P0.01000 ( set maximum deviation from commanded toolpath )

G04 P0 ( dwell for no time -- G64 should not smooth over this point )
G53 G00 Z-2.0 ( retract )
`
	drillDown = `G00 Z2.00000
G01 Z%f F100.00000
G01 Z0.50000 F100.00000
G00 Z2.00000 ( retract )
`
	switchTool = `
(MSG, Change tool bit to drill size %f mm)
M0      (Temporary machine stop.)
M3      (Spindle on clockwise.)
`

	retract = `G53 G00 Z-2.0
G53 G00 X-189.0
M5      (Spindle stop.)
`
)

type coordinates struct {
	x float64
	y float64
}

func (c coordinates) String() string {
	return fmt.Sprintf("X % 7.2f      Y % 7.2f", c.x, c.y)
}

type aperture struct {
	name string
	size float64
	pos  []coordinates
}

type BySize []aperture

func (a BySize) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].size < a[j].size }

func (a aperture) String() string {
	pos := ""
	for _, e := range a.pos {
		pos += fmt.Sprintf("%s\n", e.String())
	}

	return fmt.Sprintf(`name: %s
size: %.2f
occurences: %d
%s`, a.name, a.size, len(a.pos), pos)
}

type drill struct {
	size float64
	pos  []coordinates
}

var (
	apertures       []aperture
	currentAperture *aperture
	decimalDivider  = 1e6
	units           = "imperial"
	drills          = []drill{
		drill{size: .300},
		drill{size: .400},
		drill{size: .500},
		drill{size: .600},
		drill{size: .700},
		drill{size: .800},
		drill{size: .900},
		drill{size: 1.000},
		drill{size: 1.100},
		drill{size: 1.200},
	}
	bottom bool
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if len(os.Args) < 2 {
		log.Fatal("usage: ", os.Args[0], " inputFile")
	}

	inputFile := os.Args[1]
	outputFile := calcOutput(inputFile)

	if bottom {
		fmt.Println("creating stencil for bottom side")
	} else {
		fmt.Println("creating stencil for top side")
	}
	fmt.Println("writing to", outputFile)

	input, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		parseLine(line)
	}

	sort.Sort(BySize(apertures))

	output, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	output.Write([]byte(outputHeader))

	for _, a := range apertures {
		categorize(a)
	}

	for _, e := range drills {
		if len(e.pos) < 1 {
			continue
		}

		e.pos = optimizePath(e.pos)

		toolSwitchMessage := fmt.Sprintf(switchTool, e.size)
		output.Write([]byte(toolSwitchMessage))
		for _, p := range e.pos {
			line := fmt.Sprintf("G00 X%f Y%f\n", p.x, p.y)
			output.Write([]byte(line))
			drillDepth := -(e.size/2 + .5)
			drillCommand := fmt.Sprintf(drillDown, drillDepth)
			output.Write([]byte(drillCommand))
		}
		output.Write([]byte(retract))
	}
}

func calcOutput(in string) string {
	bottom = strings.Contains(in, "-B.")
	return strings.Split(in, ".")[0] + outputExtension
}

var (
	parseContour           = false
	minX, maxX, minY, maxY float64
)

func parseLine(line string) {
	if parseContour {
		switch {
		case strings.HasPrefix(line, "X"):
			pos := parseCoordinates(line)
			minX = math.Min(minX, pos.x)
			maxX = math.Max(maxX, pos.x)
			minY = math.Min(minY, pos.y)
			maxY = math.Max(maxY, pos.y)
		case strings.HasPrefix(line, "G37*"):
			x := (minX + maxX) / 2
			y := (minY + maxY) / 2
			pos := coordinates{x: x, y: y}
			currentAperture.pos = append(currentAperture.pos, pos)

			parseContour = false
		}
		return
	}

	switch {
	case strings.HasPrefix(line, "%AD"):
		a := parseAperture(line)
		apertures = append(apertures, a)
	case strings.HasPrefix(line, "D"):
		name := strings.TrimSuffix(line, "*")
		currentAperture = apertureByName(name)
	case strings.HasPrefix(line, "X"):
		pos := parseCoordinates(line)
		currentAperture.pos = append(currentAperture.pos, pos)
	case strings.HasPrefix(line, "G04 Gerber Fmt "):
		parseFormat(line)
	case strings.HasPrefix(line, "G36*"):
		minX = 1e6
		maxX = -1e6
		minY = 1e6
		maxY = -1e6
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
		// size = math.Min(x, y)
		size = math.Sqrt(x * y)

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

func categorize(a aperture) {
	rf := math.Inf(1)
	var useDrill *drill

	for i, e := range drills {
		r := math.Abs(e.size - a.size)
		if r < rf {
			rf = r
			useDrill = &drills[i]
		}
	}
	useDrill.pos = append(useDrill.pos, a.pos...)
}

func optimizePath(in []coordinates) []coordinates {
	out := []coordinates{}
	outDist := 1e18

	for iteration := 0; iteration < iterationCount; iteration++ {

		rand.Seed(time.Now().UnixNano())
		i := rand.Intn(len(in))
		in[0], in[i] = in[i], in[0]

		dist := 0.0

		for i := 0; i < len(in)-2; i++ {
			minDist := 1e18
			current := &in[i]
			next := &in[i+1]
			nearest := &in[i+1]
			for j := i + 1; j < len(in); j++ {
				b := &in[j]
				dist := math.Pow(current.x-b.x, 2) + math.Pow(current.y-b.y, 2)
				if dist < minDist {
					minDist = dist
					nearest = b
				}
			}
			*next, *nearest = *nearest, *next
			dist += minDist
		}

		if dist < outDist {
			out = in
			outDist = dist
		}
	}
	return out
}
