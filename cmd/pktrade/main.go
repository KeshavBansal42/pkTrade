package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
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

func usage() {
	fmt.Print(`pktrade - same-generation Pokémon save trading

Usage:
  pktrade list <save> [<save> ...]
  pktrade preview <saveA> <saveB> <slotA> <slotB>
  pktrade trade <saveA> <saveB> <slotA> <slotB> [--yes]

Slots:
  party:N       party slot N (1-6)
  box:B/S       box B, slot S (1-based)

Trades are only allowed between saves of the same generation and the same
save format. Output files are written to ./out; input files are never
modified. A verified backup is created before writing.

Examples:
  pktrade list firered.sav
  pktrade preview firered.sav leafgreen.sav party:1 box:3/12
  pktrade trade firered.sav leafgreen.sav box:1/1 box:1/1 --yes
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	args := os.Args[2:]
	yes := false
	filtered := args[:0]
	for _, arg := range args {
		if arg == "--yes" || arg == "-y" {
			yes = true
			continue
		}
		filtered = append(filtered, arg)
	}
	args = filtered

	switch os.Args[1] {
	case "list":
		if len(args) < 1 {
			fatal("list needs at least one save file")
		}
		for _, path := range args {
			if err := listSave(path); err != nil {
				fatal("%v", err)
			}
		}
	case "preview":
		if len(args) != 4 {
			fatal("preview needs <saveA> <saveB> <slotA> <slotB>")
		}
		if err := previewCommand(args[0], args[1], args[2], args[3]); err != nil {
			fatal("%v", err)
		}
	case "trade":
		if len(args) != 4 {
			fatal("trade needs <saveA> <saveB> <slotA> <slotB>")
		}
		if err := tradeCommand(args[0], args[1], args[2], args[3], yes); err != nil {
			fatal("%v", err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func load(path string) (core.SaveFile, error) {
	before, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	save, err := core.LoadSaveFromBytes(before)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if v, ok := save.(core.Validator); ok {
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return save, nil
}

func listSave(path string) error {
	save, err := load(path)
	if err != nil {
		return err
	}
	info := core.GameInfo{Generation: save.Generation()}
	if p, ok := save.(core.InfoProvider); ok {
		info = p.Info()
	}
	fmt.Printf("=== %s ===\n", path)
	activeSlot := info.ActiveSlot
	if activeSlot == "" {
		activeSlot = "n/a"
	}
	fmt.Printf("Generation: %d | Game: %s | Family: %s | Save slot: %s\n",
		info.Generation, info.Game, info.Family, activeSlot)
	if p, ok := save.(core.TrainerProvider); ok {
		trainer, err := p.TrainerInfo()
		if err == nil {
			fmt.Printf("Trainer: %s | TID: %05d | SID: %05d\n", trainer.Name, trainer.TID, trainer.SID)
		}
	}
	if caps, ok := save.(core.CapabilityProvider); ok {
		c := caps.Capabilities()
		fmt.Printf("Supports: party=%v boxes=%v evolution=%v dex=%v\n", c.Party, c.Boxes, c.Evolution, c.Dex)
	}

	store, ok := save.(core.Storage)
	if !ok {
		party, err := save.GetParty()
		if err != nil {
			return fmt.Errorf("%s: reading party: %w", path, err)
		}
		for i, p := range party {
			fmt.Printf("  party:%d %-12s species=%d level=%d OT=%s\n",
				i+1, p.GetNickname(), p.GetSpecies(), p.GetLevel(), p.GetTrainerName())
		}
		fmt.Println()
		return nil
	}
	entries, err := store.ListSlots(false)
	if err != nil {
		return fmt.Errorf("%s: listing slots: %w", path, err)
	}
	for _, entry := range entries {
		p := entry.Pokemon
		national := ""
		if n, ok := p.(interface{ NationalID() uint16 }); ok {
			national = fmt.Sprintf(" national=%d", n.NationalID())
		}
		fmt.Printf("  %-12s %-12s species=%d level=%d item=%d OT=%s%s\n",
			entry.Ref, p.GetNickname(), p.GetSpecies(), p.GetLevel(), p.GetHeldItem(), p.GetTrainerName(), national)
	}
	fmt.Println()
	return nil
}

func previewCommand(pathA, pathB, slotA, slotB string) error {
	refA, err := core.ParseSlot(slotA)
	if err != nil {
		return err
	}
	refB, err := core.ParseSlot(slotB)
	if err != nil {
		return err
	}
	saveA, err := load(pathA)
	if err != nil {
		return err
	}
	saveB, err := load(pathB)
	if err != nil {
		return err
	}
	return printPreview(core.PreviewTrade(saveA, saveB, refA, refB))
}

func printPreview(preview *core.TradePreview) error {
	fmt.Println(preview.Message)
	for _, evo := range preview.Evolutions {
		fmt.Printf("Evolution: %s -> %s", evo.FromName, evo.ToName)
		if evo.ItemName != "" {
			fmt.Printf(" (uses %s)", evo.ItemName)
		}
		fmt.Println()
	}
	for _, warning := range preview.Warnings {
		fmt.Println("Warning:", warning)
	}
	if !preview.Supported {
		return fmt.Errorf("trade is not supported")
	}
	return nil
}

func tradeCommand(pathA, pathB, slotA, slotB string, assumeYes bool) error {
	if filepath.Clean(pathA) == filepath.Clean(pathB) {
		return fmt.Errorf("cannot trade a save file with itself")
	}
	refA, err := core.ParseSlot(slotA)
	if err != nil {
		return err
	}
	refB, err := core.ParseSlot(slotB)
	if err != nil {
		return err
	}
	saveA, err := load(pathA)
	if err != nil {
		return err
	}
	saveB, err := load(pathB)
	if err != nil {
		return err
	}
	preview := core.PreviewTrade(saveA, saveB, refA, refB)
	if err := printPreview(preview); err != nil {
		return err
	}
	if !assumeYes {
		fmt.Print("Proceed with this trade? [y/N] ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := backupFile(pathA); err != nil {
		return err
	}
	if err := backupFile(pathB); err != nil {
		return err
	}

	result, err := core.ExecuteTrade(saveA, saveB, refA, refB)
	if err != nil {
		return err
	}
	for _, evo := range result.Evolutions {
		fmt.Printf("Evolved: %s -> %s\n", evo.FromName, evo.ToName)
	}
	for _, warning := range result.Warnings {
		fmt.Println("Warning:", warning)
	}

	if err := os.MkdirAll("out", 0o755); err != nil {
		return err
	}
	outA := outputPath(pathA)
	outB := outputPath(pathB)
	if err := writeFileAtomic(outA, result.SaveA); err != nil {
		return fmt.Errorf("writing %s: %w", outA, err)
	}
	if err := writeFileAtomic(outB, result.SaveB); err != nil {
		return fmt.Errorf("writing %s: %w", outB, err)
	}

	// A fresh load of each output is the final gate. ExecuteTrade already does
	// this in memory; this catches a filesystem-level truncation too.
	if _, err := load(outA); err != nil {
		return fmt.Errorf("verifying %s: %w", outA, err)
	}
	if _, err := load(outB); err != nil {
		return fmt.Errorf("verifying %s: %w", outB, err)
	}
	fmt.Printf("\nTrade complete. Written and verified:\n  %s\n  %s\n", outA, outB)
	return nil
}

func outputPath(input string) string {
	ext := filepath.Ext(input)
	base := strings.TrimSuffix(filepath.Base(input), ext)
	return filepath.Join("out", fmt.Sprintf("%s_traded%s", base, ext))
}

func backupFile(src string) error {
	if err := os.MkdirAll("backup", 0o755); err != nil {
		return fmt.Errorf("creating backup directory: %w", err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s for backup: %w", src, err)
	}
	sum := sha256.Sum256(data)
	timestamp := time.Now().Format("20060102-150405.000")
	dest := filepath.Join("backup", fmt.Sprintf("%s.%s.%s.bak",
		filepath.Base(src), timestamp, fmt.Sprintf("%x", sum[:4])))
	if err := writeFileAtomic(dest, data); err != nil {
		return fmt.Errorf("writing backup %s: %w", dest, err)
	}
	check, err := os.ReadFile(dest)
	if err != nil {
		return fmt.Errorf("reading backup %s: %w", dest, err)
	}
	checkSum := sha256.Sum256(check)
	if checkSum != sum {
		return fmt.Errorf("backup verification failed for %s", dest)
	}
	fmt.Printf("Backed up %s -> %s\n", src, dest)
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
