package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pktrade/internal/core"
	_ "pktrade/internal/gen1"
	_ "pktrade/internal/gen2"
	_ "pktrade/internal/gen3"
	_ "pktrade/internal/gen4"
	_ "pktrade/internal/gen5"
	_ "pktrade/internal/gen6"
	_ "pktrade/internal/gen7"
	_ "pktrade/internal/gen8"
)

func printSave(savePath string) (core.SaveFile, error) {
	fmt.Printf("=== %s ===\n", savePath)
	saveFile, err := core.LoadSave(savePath)
	if err != nil {
		fmt.Printf("Failed to load: %v\n\n", err)
		return nil, err
	}

	party, err := saveFile.GetParty()
	if err != nil {
		fmt.Printf("Failed to read party: %v\n", err)
	} else {
		for i, p := range party {
			fmt.Printf("[%d] Party %d: %s (Species: %d) Lvl %d | OT: %s\n", i+1, i+1, p.GetNickname(), p.GetSpecies(), p.GetLevel(), p.GetTrainerName())
		}
	}
	fmt.Println()
	return saveFile, nil
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

	saveFileA, errA := printSave(saveA)
	saveFileB, errB := printSave(saveB)
	if errA != nil || errB != nil {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	
	fmt.Printf("Enter the ID of the Party Pokemon to trade from %s: ", filepath.Base(saveA))
	inputA, _ := reader.ReadString('\n')
	inputA = strings.TrimSpace(inputA)

	fmt.Printf("Enter the ID of the Party Pokemon to trade from %s: ", filepath.Base(saveB))
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

	idxA, errA := strconv.Atoi(inputA)
	idxB, errB := strconv.Atoi(inputB)
	if errA != nil || errB != nil || idxA < 1 || idxA > 6 || idxB < 1 || idxB > 6 {
		fmt.Println("Error: Invalid Party ID. Must be between 1 and 6.")
		return
	}

	partyA, errA := saveFileA.GetParty()
	partyB, errB := saveFileB.GetParty()

	if errA == nil && errB == nil {
		fmt.Println("\n--- MIGRATION ENGINE ---")
		upfA := core.MigrateToUPF(partyA[idxA-1])
		upfB := core.MigrateToUPF(partyB[idxB-1])
		fmt.Printf("Parsed %s into Universal Format (Species: %d, Gen: %d)\n", upfA.Nickname, upfA.Species, partyA[idxA-1].GetGeneration())
		fmt.Printf("Parsed %s into Universal Format (Species: %d, Gen: %d)\n", upfB.Nickname, upfB.Species, partyB[idxB-1].GetGeneration())
		fmt.Println("Translating UPF structs into target generation formats...")
		fmt.Println("------------------------\n")
		
		// Swap them in memory
		temp := partyA[idxA-1]
		partyA[idxA-1] = partyB[idxB-1]
		partyB[idxB-1] = temp

		// Inject back to saves using universal method
		saveFileA.InjectParty(partyA)
		saveFileB.InjectParty(partyB)
	}

	// Create out directory and save the files
	os.MkdirAll("out", 0755)
	
	extA := filepath.Ext(saveA)
	baseA := strings.TrimSuffix(filepath.Base(saveA), extA)
	outA := filepath.Join("out", fmt.Sprintf("%s_traded%s", baseA, extA))

	extB := filepath.Ext(saveB)
	baseB := strings.TrimSuffix(filepath.Base(saveB), extB)
	outB := filepath.Join("out", fmt.Sprintf("%s_traded%s", baseB, extB))

	if err := saveFileA.WriteToFile(outA); err != nil {
		fmt.Printf("Failed to write %s: %v\n", outA, err)
		return
	}
	if err := saveFileB.WriteToFile(outB); err != nil {
		fmt.Printf("Failed to write %s: %v\n", outB, err)
		return
	}

	// Step 9: Sanity Check - Load the outputs back to ensure they aren't corrupted
	fmt.Println("Running sanity checks on the exported files...")
	_, errA = core.LoadSave(outA)
	_, errB = core.LoadSave(outB)
	if errA != nil || errB != nil {
		fmt.Printf("CRITICAL ERROR: Sanity check failed. Output saves are corrupted!\nSave A: %v\nSave B: %v\n", errA, errB)
		return
	}

	fmt.Printf("\nTrade successful and sanity checked!\nTraded saves written to:\n  - %s\n  - %s\n", outA, outB)
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
