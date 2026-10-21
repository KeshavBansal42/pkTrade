package gen6

import (
	"testing"

	"pktrade/internal/core"
)

func TestMarkDexSetsCaughtAndSeen(t *testing.T) {
	save := loadFixture(t)
	pokemon, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(pokemon) == 0 {
		t.Fatal("the fixture has no party members")
	}
	record := pokemon[0]
	species := record.GetSpecies()

	base, ok := save.zukanOffset()
	if !ok {
		t.Fatal("the fixture has no Pokédex block")
	}
	bit := int(species) - 1
	caught := base + gen6Caught + bit/8
	seen := base + gen6Seen + bit/8
	if save.data[caught]&(1<<(bit&7)) != 0 {
		t.Error("the species was already marked as caught")
	}

	if err := save.MarkDex(record); err != nil {
		t.Fatalf("mark dex: %v", err)
	}
	if save.data[caught]&(1<<(bit&7)) == 0 {
		t.Error("the caught flag was not set")
	}
	if save.data[seen]&(1<<(bit&7)) == 0 {
		t.Error("the seen flag was not set")
	}
	// The language flags are marked as well, so the entry shows up in the
	// player's language. Each species owns seven bits, one per language.
	langBit := bit*gen6LangCount + 1
	lang := base + gen6LangXY + langBit/8
	if save.data[lang]&(1<<(langBit&7)) == 0 {
		t.Error("the language flag was not set")
	}
}

func TestMarkDexIgnoresForeignRecords(t *testing.T) {
	save := loadFixture(t)
	if err := save.MarkDex(nil); err == nil {
		t.Fatal("expected an error for a record that is not a Gen 6 Pokémon")
	}
}

var _ = core.Pokemon(nil)
