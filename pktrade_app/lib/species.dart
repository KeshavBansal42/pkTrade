import 'species_names.dart';

/// National Dex names, generations 1-3. The engine hands the UI a raw species
/// id, and "Species ID: 7" tells a player nothing, so the roster can name what
/// it is actually holding. Anything outside this range falls back to a number
/// rather than guessing.
const List<String> _dex = <String>[
  '',
  // Generation 1
  'BULBASAUR', 'IVYSAUR', 'VENUSAUR', 'CHARMANDER', 'CHARMELEON', 'CHARIZARD',
  'SQUIRTLE', 'WARTORTLE', 'BLASTOISE', 'CATERPIE', 'METAPOD', 'BUTTERFREE',
  'WEEDLE', 'KAKUNA', 'BEEDRILL', 'PIDGEY', 'PIDGEOTTO', 'PIDGEOT',
  'RATTATA', 'RATICATE', 'SPEAROW', 'FEAROW', 'EKANS', 'ARBOK', 'PIKACHU',
  'RAICHU', 'SANDSHREW', 'SANDSLASH', 'NIDORAN-F', 'NIDORINA', 'NIDOQUEEN',
  'NIDORAN-M', 'NIDORINO', 'NIDOKING', 'CLEFAIRY', 'CLEFABLE', 'VULPIX',
  'NINETALES', 'JIGGLYPUFF', 'WIGGLYTUFF', 'ZUBAT', 'GOLBAT', 'ODDISH',
  'GLOOM', 'VILEPLUME', 'PARAS', 'PARASECT', 'VENONAT', 'VENOMOTH',
  'DIGLETT', 'DUGTRIO', 'MEOWTH', 'PERSIAN', 'PSYDUCK', 'GOLDUCK', 'MANKEY',
  'PRIMEAPE', 'GROWLITHE', 'ARCANINE', 'POLIWAG', 'POLIWHIRL', 'POLIWRATH',
  'ABRA', 'KADABRA', 'ALAKAZAM', 'MACHOP', 'MACHOKE', 'MACHAMP', 'BELLSPROUT',
  'WEEPINBELL', 'VICTREEBEL', 'TENTACOOL', 'TENTACRUEL', 'GEODUDE',
  'GRAVELER', 'GOLEM', 'PONYTA', 'RAPIDASH', 'SLOWPOKE', 'SLOWBRO',
  'MAGNEMITE', 'MAGNETON', "FARFETCH'D", 'DODUO', 'DODRIO', 'SEEL', 'DEWGONG',
  'GRIMER', 'MUK', 'SHELLDER', 'CLOYSTER', 'GASTLY', 'HAUNTER', 'GENGAR',
  'ONIX', 'DROWZEE', 'HYPNO', 'KRABBY', 'KINGLER', 'VOLTORB', 'ELECTRODE',
  'EXEGGCUTE', 'EXEGGUTOR', 'CUBONE', 'MAROWAK', 'HITMONLEE', 'HITMONCHAN',
  'LICKITUNG', 'KOFFING', 'WEEZING', 'RHYHORN', 'RHYDON', 'CHANSEY',
  'TANGELA', 'KANGASKHAN', 'HORSEA', 'SEADRA', 'GOLDEEN', 'SEAKING', 'STARYU',
  'STARMIE', 'MR. MIME', 'SCYTHER', 'JYNX', 'ELECTABUZZ', 'MAGMAR', 'PINSIR',
  'TAUROS', 'MAGIKARP', 'GYARADOS', 'LAPRAS', 'DITTO', 'EEVEE', 'VAPOREON',
  'JOLTEON', 'FLAREON', 'PORYGON', 'OMANYTE', 'OMASTAR', 'KABUTO', 'KABUTOPS',
  'AERODACTYL', 'SNORLAX', 'ARTICUNO', 'ZAPDOS', 'MOLTRES', 'DRATINI',
  'DRAGONAIR', 'DRAGONITE', 'MEWTWO', 'MEW',
  // Generation 2
  'CHIKORITA', 'BAYLEEF', 'MEGANIUM', 'CYNDAQUIL', 'QUILAVA', 'TYPHLOSION',
  'TOTODILE', 'CROCONAW', 'FERALIGATR', 'SENTRET', 'FURRET', 'HOOTHOOT',
  'NOCTOWL', 'LEDYBA', 'LEDIAN', 'SPINARAK', 'ARIADOS', 'CROBAT', 'CHINCHOU',
  'LANTURN', 'PICHU', 'CLEFFA', 'IGGLYBUFF', 'TOGEPI', 'TOGETIC', 'NATU',
  'XATU', 'MAREEP', 'FLAAFFY', 'AMPHAROS', 'BELLOSSOM', 'MARILL', 'AZUMARILL',
  'SUDOWOODO', 'POLITOED', 'HOPPIP', 'SKIPLOOM', 'JUMPLUFF', 'AIPOM',
  'SUNKERN', 'SUNFLORA', 'YANMA', 'WOOPER', 'QUAGSIRE', 'ESPEON', 'UMBREON',
  'MURKROW', 'SLOWKING', 'MISDREAVUS', 'UNOWN', 'WOBBUFFET', 'GIRAFARIG',
  'PINECO', 'FORRETRESS', 'DUNSPARCE', 'GLIGAR', 'STEELIX', 'SNUBBULL',
  'GRANBULL', 'QWILFISH', 'SCIZOR', 'SHUCKLE', 'HERACROSS', 'SNEASEL',
  'TEDDIURSA', 'URSARING', 'SLUGMA', 'MAGCARGO', 'SWINUB', 'PILOSWINE',
  'CORSOLA', 'REMORAID', 'OCTILLERY', 'DELIBIRD', 'MANTINE', 'SKARMORY',
  'HOUNDOUR', 'HOUNDOOM', 'KINGDRA', 'PHANPY', 'DONPHAN', 'PORYGON2',
  'STANTLER', 'SMEARGLE', 'TYROGUE', 'HITMONTOP', 'SMOOCHUM', 'ELEKID',
  'MAGBY', 'MILTANK', 'BLISSEY', 'RAIKOU', 'ENTEI', 'SUICUNE', 'LARVITAR',
  'PUPITAR', 'TYRANITAR', 'LUGIA', 'HO-OH', 'CELEBI',
  // Generation 3
  'TREECKO', 'GROVYLE', 'SCEPTILE', 'TORCHIC', 'COMBUSKEN', 'BLAZIKEN',
  'MUDKIP', 'MARSHTOMP', 'SWAMPERT', 'POOCHYENA', 'MIGHTYENA', 'ZIGZAGOON',
  'LINOONE', 'WURMPLE', 'SILCOON', 'BEAUTIFLY', 'CASCOON', 'DUSTOX', 'LOTAD',
  'LOMBRE', 'LUDICOLO', 'SEEDOT', 'NUZLEAF', 'SHIFTRY', 'TAILLOW', 'SWELLOW',
  'WINGULL', 'PELIPPER', 'RALTS', 'KIRLIA', 'GARDEVOIR', 'SURSKIT',
  'MASQUERAIN', 'SHROOMISH', 'BRELOOM', 'SLAKOTH', 'VIGOROTH', 'SLAKING',
  'NINCADA', 'NINJASK', 'SHEDINJA', 'WHISMUR', 'LOUDRED', 'EXPLOUD',
  'MAKUHITA', 'HARIYAMA', 'AZURILL', 'NOSEPASS', 'SKITTY', 'DELCATTY',
  'SABLEYE', 'MAWILE', 'ARON', 'LAIRON', 'AGGRON', 'MEDITITE', 'MEDICHAM',
  'ELECTRIKE', 'MANECTRIC', 'PLUSLE', 'MINUN', 'VOLBEAT', 'ILLUMISE',
  'ROSELIA', 'GULPIN', 'SWALOT', 'CARVANHA', 'SHARPEDO', 'WAILMER', 'WAILORD',
  'NUMEL', 'CAMERUPT', 'TORKOAL', 'SPOINK', 'GRUMPIG', 'SPINDA', 'TRAPINCH',
  'VIBRAVA', 'FLYGON', 'CACNEA', 'CACTURNE', 'SWABLU', 'ALTARIA', 'ZANGOOSE',
  'SEVIPER', 'LUNATONE', 'SOLROCK', 'BARBOACH', 'WHISCASH', 'CORPHISH',
  'CRAWDAUNT', 'BALTOY', 'CLAYDOL', 'LILEEP', 'CRADILY', 'ANORITH', 'ARMALDO',
  'FEEBAS', 'MILOTIC', 'CASTFORM', 'KECLEON', 'SHUPPET', 'BANETTE', 'DUSKULL',
  'DUSCLOPS', 'TROPIUS', 'CHIMECHO', 'ABSOL', 'WYNAUT', 'SNORUNT', 'GLALIE',
  'SPHEAL', 'SEALEO', 'WALREIN', 'CLAMPERL', 'HUNTAIL', 'GOREBYSS',
  'RELICANTH', 'LUVDISC', 'BAGON', 'SHELGON', 'SALAMENCE', 'BELDUM', 'METANG',
  'METAGROSS', 'REGIROCK', 'REGICE', 'REGISTEEL', 'LATIAS', 'LATIOS', 'KYOGRE',
  'GROUDON', 'RAYQUAZA', 'JIRACHI', 'DEOXYS',
];

/// Generation 1 stores species as a scrambled index number (Bulbasaur is 153),
/// with the slots that hold MissingNo. resolving to 0 here. Index 0 is unused.
const List<int> _gen1Index = <int>[
  0, // unused
  112, 115, 32, 35, 21, 100, 34, 80, 2, 103, 108, 102, 88, 94, 29, 31,
  104, 111, 131, 59, 151, 130, 90, 72, 92, 123, 120, 9, 127, 114,
  0, 0, // MissingNo (Scizor, Shuckle)
  58, 95, 22, 16, 79, 64, 75, 113, 67, 122, 106, 107, 24, 47, 54, 96, 76,
  0, // MissingNo (Heracross)
  126,
  0, // MissingNo (Ho-Oh)
  125, 82, 109,
  0, // MissingNo (Sneasel)
  56, 86, 50, 128,
  0, 0, 0, // MissingNo (Teddiursa, Ursaring, Slugma)
  83, 48, 149,
  0, 0, 0, // MissingNo (Magcargo, Swinub, Piloswine)
  84, 60, 124, 146, 144, 145, 132, 52, 98,
  0, 0, 0, // MissingNo (Corsola, Remoraid, Octillery)
  37, 38, 25, 26,
  0, 0, // MissingNo (Delibird, Mantine)
  147, 148, 140, 141, 116, 117,
  0, 0, // MissingNo (Skarmory, Houndour)
  27, 28, 138, 139, 39, 40, 133, 136, 135, 134, 66, 41, 23, 46, 61, 62,
  13, 14, 15,
  0, // MissingNo (Houndoom)
  85, 57, 51, 49, 87,
  0, 0, // MissingNo (Kingdra, Phanpy)
  10, 11, 12, 68,
  0, // MissingNo (Donphan)
  55, 97, 42, 150, 143, 129,
  0, 0, // MissingNo (Porygon2, Stantler)
  89,
  0, // MissingNo (Smeargle)
  99, 91,
  0, // MissingNo (Tyrogue)
  101, 36, 110, 53, 105,
  0, // MissingNo (Hitmontop)
  93, 63, 65, 17, 18, 121, 1, 3, 73,
  0, // MissingNo (Smoochum)
  118, 119,
  0, 0, 0, 0, // MissingNo (Elekid, Magby, Miltank, Blissey)
  77, 78, 19, 20, 33, 30, 74, 137, 142,
  0, // MissingNo (Raikou)
  81,
  0, 0, // MissingNo (Entei, Suicune)
  4, 7, 5, 8, 6,
  0, 0, 0, 0, // MissingNo (Larvitar, Pupitar, Tyranitar, Lugia)
  43, 44, 45, 69, 70, 71, // Victreebel (last valid Gen 1 index)
];

/// Converts the species id stored in a save into a National Dex number.
///
/// Generations 1 and 2 both index species directly in the record, but Gen 1
/// uses a scrambled index (with MissingNo. gaps) while Gen 2 is in National
/// Dex order. Generation 3 inserts 25 placeholder slots after Johto, so stored
/// ids from 277 up sit 25 ahead of the National Dex. Ids with no National
/// number resolve to null.
int? toNationalId(int storedSpecies, int generation) {
  switch (generation) {
    case 1:
      if (storedSpecies < 1 || storedSpecies >= _gen1Index.length) return null;
      final national = _gen1Index[storedSpecies];
      return national == 0 ? null : national;
    case 2:
      if (storedSpecies < 1 || storedSpecies > 251) return null;
      return storedSpecies;
    case 3:
      if (storedSpecies >= 277) return storedSpecies - 25;
      if (storedSpecies >= 252) return null;
      return storedSpecies;
    default:
      return storedSpecies;
  }
}

String? speciesNameFor(int storedSpecies, int generation) {
  final national = toNationalId(storedSpecies, generation);
  if (national == null) return null;
  if (generation <= 3) {
    if (national < 1 || national >= _dex.length) return null;
    return _dex[national];
  }
  // Generations 4 and later store the National Dex number directly.
  if (national < 1 || national >= modernDex.length) return null;
  return modernDex[national];
}

/// Readable species label: the real name when known, otherwise an honest
/// placeholder that still identifies the slot.
String speciesLabel(int storedSpecies, int generation) {
  final name = speciesNameFor(storedSpecies, generation);
  if (name != null) return name;
  final national = toNationalId(storedSpecies, generation);
  return national == null ? 'NO. ???' : 'NO. $national';
}
