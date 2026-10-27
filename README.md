# pkTrade

pkTrade trades Pokémon between two save files. Two saves of the **same
generation and format family** exchange records byte-for-byte and losslessly.
FireRed/LeafGreen, Ruby/Sapphire/Emerald, and Diamond/Pearl/HeartGold/SoulSilver
are separate formats. Two saves of **different generations** (Gen 1 through
Gen 8) are exchanged by converting each record into the other game's format;
anything the destination cannot represent is reported as an explicit data loss
in the preview, and the user accepts it before anything is written. A trade
between incompatible saves is refused before anything is written.

Every supported generation can trade in the party **and** in the boxes.

## Supported generations

| Generation | Games | Party | Boxes | Trade evolutions | Pokédex write |
|---|---|---|---|---|---|
| 1 | Red / Blue / Yellow | yes | 12 boxes × 20 | yes | yes |
| 2 | Gold / Silver / Crystal | yes | 14 boxes × 20 | yes (item evolutions) | yes |
| 3 | FireRed / LeafGreen | yes | 14 boxes × 30 | yes (item evolutions) | yes |
| 4 | Diamond / Pearl, Platinum, HeartGold / SoulSilver | yes | 18 boxes × 30 | yes (item evolutions) | yes |
| 5 | Black / White, Black 2 / White 2 | yes | 24 boxes × 30 | yes (item evolutions) | yes |
| 6 | X / Y, Omega Ruby / Alpha Sapphire | yes | 31 boxes × 30 | yes (item evolutions) | yes |
| 7 | Sun / Moon, Ultra Sun / Ultra Moon | yes | 32 boxes × 30 | yes (item evolutions) | yes |
| 8 | Sword / Shield | yes | 32 boxes × 30 | yes (item evolutions) | no |

Notes:

* Gen 3 accepts FireRed and LeafGreen saves only. The engine validates the
  save size, both save slots, and the section checksums before it will read or
  write anything.
* The engine has been run against real FireRed, X/Y and Black/White saves: the
  party, the boxes, the trainer record, a party trade, a box trade and the
  Pokédex write all round-trip on those files. The other generations are
  covered by synthetic saves built from an independent implementation of each
  format.
* Gen 4 and Gen 5 both use 512 KiB save files. The formats are told apart by
  their own block checksums, so a Black/White save is never mistaken for a
  Diamond/Pearl save.
* Gen 4 and Gen 5 records use different item ids for the same item. The
  evolution tables are per generation, so a King's Rock in Gen 4 and a King's
  Rock in Gen 6 are both understood.
* Every generation numbers its items on its own list: a King's Rock is item 82
  in Gen 2, 187 in Gen 3 and 221 from Gen 4 on. The tables are per generation so
  a mixed-up number cannot silently disable an evolution.
* The Pokédex is written for Gen 1 to Gen 7. Gen 8 keeps its dex as one record
  per species with its own index table, and that table is not reproduced here,
  so a Sword/Shield dex is left untouched instead of being written with a
  guessed offset.
* Eggs, records with a broken checksum and empty slots are refused.

## Cross-generation transfers

Two saves of different generations exchange Pokémon through a neutral record
that both formats are decoded into and encoded from. The converter preserves
everything the destination can represent and lists the rest as losses:

* **Species** is carried by National Dex number. A species that does not exist
  in the destination generation blocks the transfer.
* **Level** is recomputed from the experience and the destination growth rate.
* **IVs and DVs** are converted both ways (`DV = IV >> 1`); the Gen 1/2 Special
  stat is shared and the low bit of each stat is lost when moving back.
* **EVs and stat experience** are converted so the destination contributes the
  same stat, and are clamped to the destination cap.
* **Personality, nature, gender, shininess and ability** ride on the personality
  value when both generations have one. Coming from Gen 1/2 a personality value
  is built to match the nature, gender and shininess, and a Secret ID is
  invented; going back to Gen 1/2 the gender and shininess are encoded into the
  DVs.
* **Moves and PP** keep their ids (they have been stable since Gen 1) and are
  dropped only when the move does not exist in the destination. **Held items**
  are remapped by name and dropped when absent. **Capture balls** survive where
  the destination stores them.
* **Nickname and OT name** are re-encoded into the destination charset and
  truncated to its field width.

Eggs and unrepresentable species are always blocked, never silently altered.
Every dropped or changed field appears in the preview's loss list (the app's
`ACCEPT DATA LOSS` button).

## Trade evolutions

An evolution is applied when the received Pokémon is traded, and the item it
needs is consumed. The itemless evolutions are Kadabra, Machoke, Graveler,
Haunter and (from Gen 2 on) Hitmonlee; Gen 4 adds Nosepass and the item
evolutions, Gen 5 adds Boldore and Gurdurr, Gen 6 adds Phantump, Pumpkaboo,
Spritzee and Swirlix.

| Evolution | Item | Generations |
|---|---|---|
| Poliwhirl → Politoed | King's Rock | 2+ |
| Slowpoke → Slowking | King's Rock | 2+ |
| Onix → Steelix | Metal Coat | 2+ |
| Scyther → Scizor | Metal Coat | 2+ |
| Seadra → Kingdra | Dragon Scale | 2+ |
| Porygon → Porygon2 | Up-Grade | 2+ |
| Clamperl → Huntail / Gorebyss | Deep Sea Tooth / Deep Sea Scale | 3+ |
| Porygon2 → Porygon-Z | Dubious Disc | 4+ |
| Dusclops → Dusknoir | Reaper Cloth | 4+ |
| Rhydon → Rhyperior | Protector | 4+ |
| Electabuzz → Electivire | Electirizer | 4+ |
| Magmar → Magmortar | Magmarizer | 4+ |
| Feebas → Milotic | Prism Scale | 5+ |
| Spritzee → Aromatisse | Sachet | 6+ |
| Swirlix → Slurpuff | Whipped Dream | 6+ |

One evolution is not implemented: Karrablast and Shelmet evolve into each
other when they are traded for one another, which needs both Pokémon at once.
The engine's evolution hook only sees the incoming record, so this one case is
left alone.

## Slot syntax

Every command and every API call uses the same slot syntax:

```
party:3      party slot 3 (1-6)
box:2/11     box 2, slot 11 (1-based)
```

## Command line

```
pktrade list <save> [<save> ...]
pktrade preview <saveA> <saveB> <slotA> <slotB>
pktrade trade <saveA> <saveB> <slotA> <slotB> [--yes]
```

`list` prints the save's generation, game, trainer, supported features and
every occupied slot. `preview` explains whether a trade is possible without
touching a file. `trade` writes the result to `./out`, never to the input
files, and takes a verified backup of each input first.

```bash
go run ./cmd/pktrade list "firered.sav" "leafgreen.sav"
go run ./cmd/pktrade preview "firered.sav" "leafgreen.sav" party:1 box:3/12
go run ./cmd/pktrade trade "firered.sav" "leafgreen.sav" box:1/1 box:1/1 --yes
```

## Mobile API

The `mobile` package is bound to Flutter with gomobile. The Flutter app talks
to it over the `com.pktrade/bridge` method channel.

| Function | Purpose |
|---|---|
| `ParsePartyJSON(bytes)` | Party of a save as JSON, including the real trainer record |
| `ListSlotsJSON(bytes, includeEmpty)` | Party and boxes of a save as JSON, with the box geometry |
| `PreviewTradeJSON(a, b, i, j)` | Preview of a party trade (the original entry point) |
| `PreviewSlotTradeJSON(a, b, slotA, slotB)` | Preview of a party or box trade |
| `PerformTrade(a, b, i, j)` | Party trade, returns both rewritten saves |
| `PerformSlotTrade(a, b, slotA, slotB)` | Party or box trade, returns both rewritten saves |

The JSON contract never changed: `supported`, `lossy`, `message` and `losses`
keep their lower-camel-case names. For a same-generation trade `lossy` is
`false` and `losses` is empty; a cross-generation transfer sets `lossy` and
lists every field the destination cannot keep. `supported` remains the only
field a caller may use to decide whether a trade can run.

The app reaches the boxes through `ListSlotsJSON`: the storage screen lists
every box of a save, and the slot reference of the chosen Pokémon (`party:3`,
`box:2/11`) is what the preview and the trade use.

## The app

The interface is a handheld console: a plum plastic shell with a moulded lip, a
dark bezel sunk into it, a bright LCD with a faint scan pattern, and real
controls along the chin.

* **The console works.** The D-pad walks the party list, A picks the
  highlighted Pokémon, B opens the storage and START sends the trade. The
  keyboard mirrors it: the arrows walk, enter picks, escape opens the storage
  and space sends the trade. The shell gives way to fit a narrow phone: the
  speaker and the wide pills step aside under 360 points.
* **The table** — the two saves face each other with a cable between them. Each
  side lists its party, every row carrying the creature's dex number and the
  ball it was caught in. The selected row names the slot it came from.
* **Storage** — `BOXES` opens the save's boxes as a full screen, one box at a
  time, the way the game's own PC screen takes over. It slides in over the
  table and the chosen Pokémon drops straight into the trade.
* **The transfer check** — both creatures side by side with their balls, their
  slots, and any evolution the trade will trigger, before anything is written.
* **The trade** — the two balls rise from their trainer's side, cross, and land
  on the other side with a flash, and the result names the slots that changed.
  Tapping anywhere skips to the end.
* **The boot** — the panel lights up with a sweep of light when the machine
  wakes.

Motion answers input: the only unrequested movement is the power-on. Everything
else — the row highlight, the storage page, the notice, the ball swap, the
button presses — answers a tap, and the whole app collapses to instant state
changes when the platform asks for reduced motion.

The ball artwork is the official Pokémon item art. The capture ball is read
out of the record — the original-trainer block in Gen 4 to Gen 8 and the
origins block in Gen 3. Gen 1 and Gen 2 do not store a ball at all, so those
trades show no ball rather than a wrong one.

## Ball sprites

The capture ball of a traded Pokémon is read out of the record: the ball byte
of the original-trainer block in Gen 4 to Gen 8, and the origins block in Gen 3.
Gen 1 and Gen 2 have no capture ball in the save, so those trades show the
unknown marker.

The item numbers differ per generation, and the sprite table follows the
numbers the games use: the twelve classic balls are 1 to 12 in every
generation, Dusk, Heal, Quick and Cherish Ball are 13 to 16 from Gen 4 on, the
apricorn balls are 492 to 500, the Dream Ball is 576 and the Beast Ball is 851.

The ball artwork in `pktrade_app/assets/images/balls` is the official Pokémon
item art, cropped to the 18 by 18 pixel icon the app draws.

## Building the Android library

The Flutter app loads the Go engine from `pktrade_app/android/app/libs/mobile.aar`.
That file is a build artifact and is not stored in the repository, so build it
before assembling the app:

```bash
make aar          # gomobile bind -> pktrade_app/android/app/libs/mobile.aar
make apk          # debug APK
```

`make aar` needs the Android SDK and NDK; point `ANDROID_HOME` at the SDK
directory if it is not already set.

## Tests

```bash
make test         # go test ./...
make check        # go vet, go test, flutter analyze, flutter test
```

The Go tests use synthetic save files that were generated with an independent
implementation of each save format, so the engine is checked against the
format and not only against itself. The fixtures live in the `testdata`
directory of each generation package.
