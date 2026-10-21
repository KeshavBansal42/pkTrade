import 'species.dart';

/// One party member as the engine reports it. The engine reads the save's own
/// trainer record, so the player identity comes from the save and the per
/// record OT name stays available for comparison.
class PartyPokemon {
  const PartyPokemon({
    required this.index,
    required this.species,
    required this.level,
    required this.ballId,
    required this.nickname,
    required this.otName,
    required this.otId,
    required this.generation,
    this.slot = '',
  });

  final int index;
  final int species;
  final int level;
  final int? ballId;
  final String nickname;
  final String otName;
  final int otId;
  final int generation;

  /// Slot reference of this record, for example `party:1` or `box:2/5`.
  final String slot;

  /// Where this record sits, in words a player would use.
  String get slotLabel {
    if (slot.isEmpty) return '';
    if (slot.startsWith('party:')) return 'PARTY ${slot.split(':').last}';
    final parts = slot.split('/');
    if (parts.length != 2) return slot.toUpperCase();
    return 'BOX ${parts[0].split(':').last}/${parts[1]}';
  }

  /// The box this record sits in, zero when it is in the party.
  int get boxNumber {
    if (slot.isEmpty || slot.startsWith('party:')) return 0;
    final parts = slot.split('/');
    if (parts.length != 2) return 0;
    return int.tryParse(parts[0].split(':').last) ?? 0;
  }

  String get speciesName => speciesLabel(species, generation);

  String get dexLabel {
    final national = toNationalId(species, generation);
    return national == null ? '???' : national.toString().padLeft(3, '0');
  }

  String get displayName {
    final n = nickname.trim();
    return n.isEmpty ? speciesName : n;
  }

  bool get isNicknamed =>
      nickname.trim().isNotEmpty &&
      nickname.trim().toUpperCase() != speciesName.toUpperCase();

  String get otIdLabel => otId.toString().padLeft(5, '0');
}

/// One occupied or empty slot of a save. Party slots and box slots share the
/// same reference syntax the engine uses: `party:3` and `box:2/11`.
class SlotEntry {
  const SlotEntry({
    required this.slot,
    required this.empty,
    this.pokemon,
  });

  final String slot;
  final bool empty;
  final PartyPokemon? pokemon;

  bool get isParty => slot.startsWith('party:');

  /// Box number of a box slot, zero for the party.
  int get box {
    if (isParty) return 0;
    final parts = slot.split('/');
    return int.tryParse(parts.first.split(':').last) ?? 0;
  }

  /// Position inside the box, zero for the party.
  int get position {
    if (isParty) return 0;
    final parts = slot.split('/');
    return int.tryParse(parts.last) ?? 0;
  }
}

/// Everything the engine knows about the slots of a save.
class SaveSlots {
  const SaveSlots({
    required this.generation,
    required this.boxCount,
    required this.boxCapacity,
    required this.slots,
  });

  final int generation;
  final int boxCount;
  final int boxCapacity;
  final List<SlotEntry> slots;

  bool get hasBoxes => boxCount > 0;

  List<SlotEntry> get boxes =>
      slots.where((s) => !s.isParty).toList(growable: false);

  List<int> get boxNumbers {
    final numbers = <int>{for (final s in boxes) s.box};
    return numbers.toList()..sort();
  }

  List<SlotEntry> box(int number) =>
      boxes.where((s) => s.box == number).toList(growable: false);

  SlotEntry? byRef(String ref) {
    for (final slot in slots) {
      if (slot.slot == ref) return slot;
    }
    return null;
  }

  factory SaveSlots.fromJson(Map<String, dynamic> json) {
    final generation = (json['generation'] as num?)?.toInt() ?? 0;
    final rawSlots = json['slots'];
    final slots = <SlotEntry>[];
    if (rawSlots is List) {
      for (final entry in rawSlots) {
        if (entry is! Map) continue;
        final map = Map<String, dynamic>.from(entry);
        final empty = map['empty'] == true;
        slots.add(
          SlotEntry(
            slot: (map['slot'] as String?) ?? '',
            empty: empty,
            pokemon: empty
                ? null
                : PartyPokemon(
                    index: (map['index'] as num?)?.toInt() ?? 0,
                    species: (map['species'] as num?)?.toInt() ?? 0,
                    level: (map['level'] as num?)?.toInt() ?? 0,
                    ballId: (map['ballId'] as num?)?.toInt(),
                    nickname: (map['nickname'] as String?)?.trim() ?? '',
                    otName: (map['otName'] as String?)?.trim() ?? '',
                    otId: (map['otId'] as num?)?.toInt() ?? 0,
                    generation: generation,
                    slot: (map['slot'] as String?) ?? '',
                  ),
          ),
        );
      }
    }
    return SaveSlots(
      generation: generation,
      boxCount: (json['boxCount'] as num?)?.toInt() ?? 0,
      boxCapacity: (json['boxCapacity'] as num?)?.toInt() ?? 0,
      slots: slots,
    );
  }
}

class TransferPreview {
  const TransferPreview({
    required this.supported,
    required this.lossy,
    required this.message,
    required this.losses,
    this.evolutions = const [],
  });

  final bool supported;
  final bool lossy;
  final String message;
  final List<String> losses;

  /// The evolutions the trade will trigger.
  final List<TradeEvolution> evolutions;

  factory TransferPreview.fromJson(Map<String, dynamic> json) =>
      TransferPreview(
        supported: json['supported'] == true,
        lossy: json['lossy'] == true,
        message:
            (json['message'] as String?) ??
            'Transfer compatibility is unknown.',
        losses: (json['losses'] as List<dynamic>? ?? const [])
            .whereType<String>()
            .toList(growable: false),
        evolutions: (json['evolutions'] as List<dynamic>? ?? const [])
            .whereType<Map>()
            .map((e) => TradeEvolution.fromJson(Map<String, dynamic>.from(e)))
            .toList(growable: false),
      );
}

/// One evolution a trade will trigger.
class TradeEvolution {
  const TradeEvolution({
    required this.from,
    required this.to,
    required this.fromName,
    required this.toName,
    this.itemName = '',
  });

  final int from;
  final int to;
  final String fromName;
  final String toName;
  final String itemName;

  String get label {
    final was = fromName.isEmpty ? '#$from' : fromName.toUpperCase();
    final now = toName.isEmpty ? '#$to' : toName.toUpperCase();
    return itemName.isEmpty ? '$was → $now' : '$was → $now  ($itemName)';
  }

  factory TradeEvolution.fromJson(Map<String, dynamic> json) =>
      TradeEvolution(
        from: (json['from'] as num?)?.toInt() ?? 0,
        to: (json['to'] as num?)?.toInt() ?? 0,
        fromName: (json['fromName'] as String?) ?? '',
        toName: (json['toName'] as String?) ?? '',
        itemName: (json['itemName'] as String?) ?? '',
      );
}

class SaveData {
  const SaveData({
    required this.generation,
    required this.trainerName,
    required this.trainerId,
    required this.party,
  });

  final int generation;
  final String trainerName;
  final int trainerId;
  final List<PartyPokemon> party;

  bool get isEmpty => party.isEmpty;

  String get trainerIdLabel => trainerId.toString().padLeft(5, '0');

  PartyPokemon? byIndex(int? index) {
    if (index == null) return null;
    for (final p in party) {
      if (p.index == index) return p;
    }
    return null;
  }

  static SaveData fromJson(Map<String, dynamic> json) {
    final generation = (json['generation'] as num?)?.toInt() ?? 3;
    final rawParty = json['party'];
    final party = <PartyPokemon>[];
    if (rawParty is List) {
      for (final entry in rawParty) {
        if (entry is! Map) continue;
        party.add(
          PartyPokemon(
            index: (entry['index'] as num?)?.toInt() ?? party.length + 1,
            species: (entry['species'] as num?)?.toInt() ?? 0,
            level: (entry['level'] as num?)?.toInt() ?? 0,
            ballId: (entry['ballId'] as num?)?.toInt(),
            nickname: (entry['nickname'] as String?)?.trim() ?? '',
            otName: (entry['otName'] as String?)?.trim() ?? '',
            otId: (entry['otId'] as num?)?.toInt() ?? 0,
            generation: generation,
            slot: (entry['slot'] as String?) ?? 'party:${party.length + 1}',
          ),
        );
      }
    }

    final first = party.isEmpty ? null : party.first;
    // Prefer the trainer record the engine read out of the save. The original
    // trainer of the first Pokémon is only a fallback for saves whose format
    // does not store a player name.
    final saveTrainerName = (json['trainerName'] as String?)?.trim() ?? '';
    final trainerName = saveTrainerName.isNotEmpty
        ? saveTrainerName
        : ((first != null && first.otName.isNotEmpty)
            ? first.otName
            : 'TRAINER');
    final saveTid = (json['tid'] as num?)?.toInt() ?? 0;
    final trainerId =
        saveTid > 0 ? saveTid : (first != null ? first.otId : 0);

    return SaveData(
      generation: generation,
      trainerName: trainerName,
      trainerId: trainerId,
      party: party,
    );
  }
}
