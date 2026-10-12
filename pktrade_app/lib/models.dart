import 'species.dart';

/// One party member as the engine reports it. The save's top-level trainer
/// fields are hardcoded placeholders in the engine, so the honest source of
/// trainer identity is each Pokemon's own OT name and id.
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
  });

  final int index;
  final int species;
  final int level;
  final int? ballId;
  final String nickname;
  final String otName;
  final int otId;
  final int generation;

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

class TransferPreview {
  const TransferPreview({
    required this.supported,
    required this.lossy,
    required this.message,
    required this.losses,
  });

  final bool supported;
  final bool lossy;
  final String message;
  final List<String> losses;

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
          ),
        );
      }
    }

    final first = party.isEmpty ? null : party.first;
    final trainerName = (first != null && first.otName.isNotEmpty)
        ? first.otName
        : ((json['trainerName'] as String?) ?? 'TRAINER');
    final trainerId = (first != null && first.otId > 0)
        ? first.otId
        : ((json['tid'] as num?)?.toInt() ?? 0);

    return SaveData(
      generation: generation,
      trainerName: trainerName,
      trainerId: trainerId,
      party: party,
    );
  }
}
