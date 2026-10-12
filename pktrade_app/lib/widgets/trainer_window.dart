import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';
import 'gba_widgets.dart';

/// One trainer's side of the link: a titled window holding their party.
///
/// Rows are deliberately uniform — same height, same plate, same type — the way
/// a handheld party menu is, so the roster reads as a list rather than as a
/// stack of cards. The only row that deviates is the one in play.
class TrainerWindow extends StatelessWidget {
  const TrainerWindow({
    super.key,
    required this.playerOne,
    required this.data,
    required this.selectedIndex,
    required this.interactive,
    required this.onSelect,
    required this.onPickFile,
  });

  final bool playerOne;
  final SaveData? data;
  final int? selectedIndex;
  final bool interactive;
  final ValueChanged<int> onSelect;
  final VoidCallback onPickFile;

  Color get _lane => playerOne ? Gba.crimson : Gba.leaf;
  String get _playerTag => playerOne ? 'P1' : 'P2';

  @override
  Widget build(BuildContext context) {
    final data = this.data;
    return GbaWindow(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _strip(data),
          Expanded(
            child: data == null
                ? _EmptyBay(
                    lane: _lane,
                    playerOne: playerOne,
                    enabled: interactive,
                    onPickFile: onPickFile,
                  )
                : _roster(context, data),
          ),
          if (data != null) ...[const HairRule(), _footer(data)],
        ],
      ),
    );
  }

  Widget _strip(SaveData? data) {
    return Container(
      color: _lane,
      padding: const EdgeInsets.fromLTRB(8, 5, 6, 5),
      child: Row(
        children: [
          TagPin(
            label: _playerTag,
            color: Gba.onColor,
            fill: Gba.ink.withValues(alpha: 0.25),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              data == null ? 'NO SAVE LOADED' : data.trainerName.toUpperCase(),
              style: GbaText.windowTitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (data != null)
            TagPin(label: 'GEN ${data.generation}', color: Gba.onColor),
        ],
      ),
    );
  }

  Widget _roster(BuildContext context, SaveData data) {
    if (data.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(10),
          child: Text(
            'THIS PARTY IS EMPTY',
            textAlign: TextAlign.center,
            style: GbaText.label,
          ),
        ),
      );
    }

    return ListView.builder(
      padding: EdgeInsets.zero,
      itemExtent: 48,
      itemCount: data.party.length,
      itemBuilder: (context, i) {
        final p = data.party[i];
        return _MonRow(
          key: ValueKey('${data.trainerId}-${p.index}'),
          pokemon: p,
          lane: _lane,
          selected: selectedIndex == p.index,
          enabled: interactive,
          onTap: () => onSelect(p.index),
        );
      },
    );
  }

  Widget _footer(SaveData data) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 5, 6, 5),
      child: Row(
        children: [
          Text('ID ${data.trainerIdLabel}', style: GbaText.tiny),
          const Spacer(),
          _ChangeAction(enabled: interactive, onTap: onPickFile),
        ],
      ),
    );
  }
}

class _MonRow extends StatelessWidget {
  const _MonRow({
    super.key,
    required this.pokemon,
    required this.lane,
    required this.selected,
    required this.enabled,
    required this.onTap,
  });

  final PartyPokemon pokemon;
  final Color lane;
  final bool selected;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final onBar = selected ? Gba.onColor : Gba.ink;
    final subOnBar = selected
        ? Gba.onColor.withValues(alpha: 0.78)
        : Gba.inkSoft;

    return Semantics(
      button: true,
      selected: selected,
      enabled: enabled,
      label:
          'Slot ${pokemon.index}, ${pokemon.speciesName}, level '
          '${pokemon.level}${selected ? ', selected' : ''}',
      child: Padding(
        padding: const EdgeInsets.fromLTRB(3, 2, 3, 2),
        child: Material(
          color: selected ? Colors.transparent : Colors.transparent,
          child: InkWell(
            onTap: enabled ? onTap : null,
            borderRadius: BorderRadius.circular(2),
            splashColor: lane.withValues(alpha: 0.14),
            highlightColor: lane.withValues(alpha: 0.10),
            focusColor: lane.withValues(alpha: 0.22),
            child: AnimatedContainer(
              duration: GbaMotion.base,
              curve: GbaMotion.settle,
              decoration: BoxDecoration(
                color: selected ? lane : Colors.transparent,
                borderRadius: BorderRadius.circular(2),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 5),
              child: Row(
                children: [
                  SizedBox(
                    width: 10,
                    child: selected
                        ? Align(
                            alignment: Alignment.centerLeft,
                            child: MenuCursor(color: Gba.onColor, size: 8),
                          )
                        : null,
                  ),
                  const SizedBox(width: 5),
                  DexPlate(label: pokemon.dexLabel, onDark: selected, size: 30),
                  const SizedBox(width: 9),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Text(
                          pokemon.isNicknamed
                              ? pokemon.displayName
                              : pokemon.speciesName,
                          style: GbaText.name.copyWith(color: onBar),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          pokemon.isNicknamed
                              ? pokemon.speciesName
                              : 'OT ${pokemon.otName}',
                          style: GbaText.tiny.copyWith(color: subOnBar),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    'LV${pokemon.level}',
                    style: GbaText.number.copyWith(color: onBar),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// An unloaded side: a cartridge bay that says exactly what to do.
class _EmptyBay extends StatelessWidget {
  const _EmptyBay({
    required this.lane,
    required this.playerOne,
    required this.enabled,
    required this.onPickFile,
  });

  final Color lane;
  final bool playerOne;
  final bool enabled;
  final VoidCallback onPickFile;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      enabled: enabled,
      label: 'Load a save file for player ${playerOne ? 1 : 2}',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: enabled ? onPickFile : null,
          splashColor: lane.withValues(alpha: 0.12),
          highlightColor: lane.withValues(alpha: 0.08),
          focusColor: lane.withValues(alpha: 0.20),
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                CartridgeGlyph(color: lane),
                const SizedBox(height: 10),
                Text('INSERT SAVE', style: GbaText.label.copyWith(color: lane)),
                const SizedBox(height: 4),
                Text('TAP TO OPEN A .SAV FILE', style: GbaText.tiny),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ChangeAction extends StatelessWidget {
  const _ChangeAction({required this.enabled, required this.onTap});

  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      enabled: enabled,
      label: 'Change this save file',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: enabled ? onTap : null,
          borderRadius: BorderRadius.circular(2),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
            child: Text(
              'CHANGE FILE',
              style: GbaText.tiny.copyWith(
                color: enabled ? Gba.ink : Gba.inkFaint,
                decoration: TextDecoration.underline,
                decorationColor: enabled ? Gba.ink : Gba.inkFaint,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
