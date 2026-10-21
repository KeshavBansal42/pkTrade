package mobile

import (
	"encoding/json"
	"strings"
	"testing"

	"pktrade/internal/testutil"
)

func TestListSlotsJSONIncludesBoxes(t *testing.T) {
	data := testutil.ReadGzip(t, "../internal/gen6/testdata/xy.bin.gz")
	raw, err := ListSlotsJSON(data, false)
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	var payload struct {
		Generation  int `json:"generation"`
		BoxCount    int `json:"boxCount"`
		BoxCapacity int `json:"boxCapacity"`
		Slots       []struct {
			Slot    string `json:"slot"`
			Empty   bool   `json:"empty"`
			Species uint16 `json:"species"`
			Level   byte   `json:"level"`
			BallID  *byte  `json:"ballId"`
		} `json:"slots"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Generation != 6 {
		t.Errorf("generation = %d, want 6", payload.Generation)
	}
	if payload.BoxCount != 31 || payload.BoxCapacity != 30 {
		t.Errorf("box geometry = %dx%d, want 31x30", payload.BoxCount, payload.BoxCapacity)
	}
	if !strings.Contains(raw, `"slot":"party:1"`) {
		t.Error("the party is missing from the slot list")
	}
	if !strings.Contains(raw, `"slot":"box:1/1"`) {
		t.Error("the boxes are missing from the slot list")
	}
}
