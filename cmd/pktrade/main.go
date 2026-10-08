package main

import (
	"fmt"
	"pktrade/internal/gen3"
)

func main() {
	saves := []string{
		"testdata/1695 - Pokemon Fire Red (U)(Independent).sav",
		"testdata/1695 - Pokemon Fire Red (U)(Independent)(1).sav",
	}

	for _, save := range saves {
		_, err := gen3.LoadSave(save)
		if err != nil {
			fmt.Printf("Failed to load %s: %v\n", save, err)
		}
		fmt.Println()
	}
}
