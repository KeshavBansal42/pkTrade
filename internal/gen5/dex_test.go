package gen5

import (
	"testing"
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
	bit := int(species) - 1
	block, ok := save.blockByName("Pokedex")
	if !ok {
		t.Fatal("the fixture has no Pokédex block")
	}
	base := block.Offset
	caught := base + gen5Caught + bit/8
	seen := base + gen5Seen + bit/8
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
}
