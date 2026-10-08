package main

import (
	"fmt"
	"os"
	"pktrade/internal/gen3"
)

func printSave(savePath string) {
	fmt.Printf("=== %s ===\n", savePath)
	saveFile, err := gen3.LoadSave(savePath)
	if err != nil {
		fmt.Printf("Failed to load: %v\n\n", err)
		return
	}

	trainer, err := gen3.GetTrainerInfo(saveFile.ActiveSlot)
	if err != nil {
		fmt.Printf("Failed to read trainer info: %v\n\n", err)
		return
	}
	fmt.Printf("Trainer: %s (TID: %05d, SID: %05d)\n\n", trainer.Name, trainer.TID, trainer.SID)

	party, err := gen3.GetParty(saveFile.ActiveSlot)
	if err != nil {
		fmt.Printf("Failed to read party: %v\n", err)
	} else {
		for i, p := range party {
			fmt.Printf("[%d] Party %d: %s (Species: %d) Lvl %d | OT: %s %05d\n", i+1, i+1, p.Nickname, p.Species, p.Level, p.OTName, p.OTID&0xFFFF)
		}
	}

	boxes, err := gen3.GetPCBoxes(saveFile.ActiveSlot)
	if err != nil {
		fmt.Printf("Failed to read PC boxes: %v\n", err)
	} else {
		offset := len(party) + 1
		for i, p := range boxes {
			fmt.Printf("[%d] Box %d / Slot %d: %s (Species: %d) Exp %d | OT: %s %05d\n", offset+i, p.BoxNum, p.BoxSlot, p.Nickname, p.Species, p.Experience, p.OTName, p.OTID&0xFFFF)
		}
	}
	fmt.Println()
}

func main() {
	if len(os.Args) < 4 || os.Args[1] != "list" {
		fmt.Println("Usage: pktrade list <save1> <save2>")
		return
	}

	printSave(os.Args[2])
	printSave(os.Args[3])
}
