package main

import (
	"fmt"
	"log"
	"os"
)

func exportGCode(file string) {
	output, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	output.Write([]byte(outputHeader))

	for _, e := range drills {
		if len(e.pos) < 1 {
			continue
		}

		// optimize toolpath for each drill
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
