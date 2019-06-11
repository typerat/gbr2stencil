package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	maxIterationCount = 1000
	outputExtension   = ".Stencil.ngc"
	outputHeader      = `G94 ( Millimeters per minute feed rate. )
G21 ( Units == Millimeters. )

G90 ( Absolute coordinates. )
S10000 ( RPM spindle speed. )
G64 P0.01000 ( set maximum deviation from commanded toolpath )

G04 P0 ( dwell for no time -- G64 should not smooth over this point )
G53 G00 Z0.0 ( retract )
`
	drillDown = `G00 Z1.00000
G01 Z%f F200.00000
G01 Z1.00000 F200.00000
G00 Z5.00000 ( retract )
`
	switchTool = `
(MSG, Change tool bit to drill size %f mm)
M0      (Temporary machine stop.)
M3      (Spindle on clockwise.)
`

	retract = `M5      (Spindle stop.)
G53 G00 Z0.0
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

	parseInput(inputFile)

	for _, a := range apertures {
		categorize(a)
	}

	exportGCode(outputFile)
}

func calcOutput(in string) string {
	bottom = strings.Contains(in, "-B.")
	return strings.Split(in, ".")[0] + outputExtension
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
	rand.Seed(time.Now().UnixNano())

	out := []coordinates{}
	outDist := 1e18

	iterationCount := maxIterationCount
	if len(in) < maxIterationCount {
		iterationCount = len(in)
	}

	// for iteration := 0; iteration < iterationCount; iteration++ {

	for iteration := 0; iteration < iterationCount; iteration++ {
		i := iteration

		// try random start points
		if len(in) > maxIterationCount {
			i = rand.Intn(len(in))
		}

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
