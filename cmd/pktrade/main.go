package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func backupFile(src string) error {
	if err := os.MkdirAll("backup", 0755); err != nil {
		return err
	}

	baseName := filepath.Base(src)
	timestamp := time.Now().Format("20060102150405")
	dest := filepath.Join("backup", fmt.Sprintf("%s.%s.bak", baseName, timestamp))

	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dest, srcData, 0644); err != nil {
		return err
	}

	destData, err := os.ReadFile(dest)
	if err != nil {
		return err
	}

	srcHash := sha256.Sum256(srcData)
	destHash := sha256.Sum256(destData)

	if srcHash != destHash {
		return fmt.Errorf("hash mismatch for %s", src)
	}

	fmt.Printf("Backed up %s -> %s\n", src, dest)
	return nil
}

func doTrade(saveA, saveB string) {
	if filepath.Clean(saveA) == filepath.Clean(saveB) {
		fmt.Println("Error: Cannot trade a save file with itself.")
		return
	}

	printSave(saveA)
	printSave(saveB)

	reader := bufio.NewReader(os.Stdin)
	
	fmt.Printf("Enter the ID of the Pokemon to trade from %s: ", filepath.Base(saveA))
	inputA, _ := reader.ReadString('\n')
	inputA = strings.TrimSpace(inputA)

	fmt.Printf("Enter the ID of the Pokemon to trade from %s: ", filepath.Base(saveB))
	inputB, _ := reader.ReadString('\n')
	inputB = strings.TrimSpace(inputB)

	fmt.Printf("\nSelected to trade Pokemon [%s] from Save A and Pokemon [%s] from Save B.\n", inputA, inputB)
	fmt.Println("Creating secure backups before proceeding...")

	if err := backupFile(saveA); err != nil {
		fmt.Printf("Backup failed for %s: %v\n", saveA, err)
		return
	}
	if err := backupFile(saveB); err != nil {
		fmt.Printf("Backup failed for %s: %v\n", saveB, err)
		return
	}

	fmt.Println("Backups verified! Ready for raw trade (Step 7).")
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage:")
		fmt.Println("  pktrade list <save1> <save2>")
		fmt.Println("  pktrade trade <save1> <save2>")
		return
	}

	cmd := os.Args[1]
	save1 := os.Args[2]
	save2 := os.Args[3]

	if cmd == "list" {
		printSave(save1)
		printSave(save2)
	} else if cmd == "trade" {
		doTrade(save1, save2)
	} else {
		fmt.Println("Unknown command:", cmd)
	}
}
