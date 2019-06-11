package main

import (
	"math"
)

func getCenter(in []coordinates) coordinates {
	minX := math.Inf(1)
	maxX := math.Inf(-1)
	minY := math.Inf(1)
	maxY := math.Inf(-1)

	for _, p := range in {
		minX = math.Min(minX, p.x)
		maxX = math.Max(maxX, p.x)
		minY = math.Min(minY, p.y)
		maxY = math.Max(maxY, p.y)
	}

	x := (minX + maxX) / 2
	y := (minY + maxY) / 2

	return coordinates{x, y}
}

// assumes rectangular pads
func getSize(in []coordinates) float64 {
	minX := math.Inf(1)
	maxX := math.Inf(-1)
	minY := math.Inf(1)
	maxY := math.Inf(-1)

	for _, p := range in {
		minX = math.Min(minX, p.x)
		maxX = math.Max(maxX, p.x)
		minY = math.Min(minY, p.y)
		maxY = math.Max(maxY, p.y)
	}

	x := maxX - minX
	y := maxY - minY

	// best size for area
	areaSize := math.Sqrt(x * y)
	// max size for smaller dimension
	maxSize := math.Min(x, y) * 1.2

	size := math.Min(maxSize, areaSize)

	return size
}
