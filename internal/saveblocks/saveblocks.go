package saveblocks

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed save_blocks.json
var raw []byte

type Block struct {
	ID       int    `json:"id"`
	Offset   int    `json:"offset"`
	Length   int    `json:"length"`
	Name     string `json:"name"`
	Checksum int    `json:"checksum"`
	Mirror   int    `json:"mirror"`
}

type Table struct {
	Size           int     `json:"size"`
	MetadataOffset int     `json:"metadata_offset"`
	Checksum       string  `json:"checksum"`
	Blocks         []Block `json:"blocks"`
}

var tables map[string]Table

func init() {
	if err := json.Unmarshal(raw, &tables); err != nil {
		panic(fmt.Sprintf("invalid embedded save block table: %v", err))
	}
}

func Get(name string) (Table, bool) {
	t, ok := tables[name]
	return t, ok
}
