import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';
import 'ball_art.dart';
import 'gba_widgets.dart';

/// The storage screen: every box of a save, one box at a time.
///
/// It replaces the table while it is open, the way the game's own PC screen
/// takes over the screen, and hands the chosen slot back to the caller.
class StorageScreen extends StatefulWidget {
  const StorageScreen({
    super.key,
    required this.slots,
    required this.trainerName,
    required this.generation,
    required this.lane,
    required this.selectedSlot,
    required this.onPick,
    required this.onClose,
  });

  final SaveSlots slots;
  final String trainerName;
  final int generation;
  final Color lane;
  final String? selectedSlot;
  final ValueChanged<String> onPick;
  final VoidCallback onClose;

  @override
  State<StorageScreen> createState() => _StorageScreenState();
}

class _StorageScreenState extends State<StorageScreen> {
  int _box = 0;

  @override
  void initState() {
    super.initState();
    final numbers = widget.slots.boxNumbers;
    if (numbers.isNotEmpty) _box = numbers.first;
  }

  @override
  Widget build(BuildContext context) {
    final slots = widget.slots;
    final numbers = slots.boxNumbers;
    final entries = slots.box(_box);
    final occupied = entries.where((e) => !e.empty).toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _header(),
        Container(
          color: Gba.shade,
          padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 4),
          child: SizedBox(
            height: 26,
            child: ListView.builder(
              scrollDirection: Axis.horizontal,
              itemCount: numbers.length,
              itemBuilder: (context, i) {
                final number = numbers[i];
                final active = number == _box;
                return Padding(
                  padding: const EdgeInsets.only(right: 3),
                  child: _BoxTab(
                    number: number,
                    active: active,
                    lane: widget.lane,
                    onTap: () => setState(() => _box = number),
                  ),
                );
              },
            ),
          ),
        ),
        Expanded(
          child: occupied.isEmpty
              ? Center(
                  child: Text('THIS BOX IS EMPTY', style: GbaText.label),
                )
              : ListView.builder(
                  padding: const EdgeInsets.fromLTRB(4, 6, 4, 6),
                  itemExtent: 40,
                  itemCount: occupied.length + 1,
                  itemBuilder: (context, i) {
                    if (i == occupied.length) {
                      final empty = slots.boxCapacity - occupied.length;
                      return Padding(
                        padding: const EdgeInsets.fromLTRB(6, 8, 0, 0),
                        child: Text(
                          empty > 0 ? '$empty EMPTY SLOTS' : '',
                          style: GbaText.tiny.copyWith(color: Gba.inkFaint),
                        ),
                      );
                    }
                    final entry = occupied[i];
                    return _SlotRow(
                      entry: entry,
                      lane: widget.lane,
                      selected: entry.slot == widget.selectedSlot,
                      onTap: () => widget.onPick(entry.slot),
                    );
                  },
                ),
        ),
        _footer(),
      ],
    );
  }

  Widget _header() {
    return Container(
      color: widget.lane,
      padding: const EdgeInsets.fromLTRB(10, 6, 8, 6),
      child: Row(
        children: [
          Text('STORAGE', style: GbaText.windowTitle),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              widget.trainerName.toUpperCase(),
              style: GbaText.tiny.copyWith(
                color: Gba.onColor.withValues(alpha: 0.85),
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          Text(
            'GEN ${widget.generation}',
            style: GbaText.tiny.copyWith(color: Gba.onColor),
          ),
        ],
      ),
    );
  }

  Widget _footer() {
    return Container(
      color: Gba.shade,
      padding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
      child: Row(
        children: [
          Text(
            _box > 0 ? 'BOX $_box OF ${widget.slots.boxCount}' : 'NO BOXES',
            style: GbaText.tiny,
          ),
          const Spacer(),
          OptionBar(label: 'CLOSE', onPressed: widget.onClose),
        ],
      ),
    );
  }
}

class _BoxTab extends StatelessWidget {
  const _BoxTab({
    required this.number,
    required this.active,
    required this.lane,
    required this.onTap,
  });

  final int number;
  final bool active;
  final Color lane;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      selected: active,
      label: 'Box $number',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(2),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 3),
            decoration: BoxDecoration(
              color: active ? lane : Gba.window,
              border: Border.all(color: Gba.ink, width: 1),
              borderRadius: BorderRadius.circular(2),
            ),
            child: Text(
              '$number',
              style: GbaText.tiny.copyWith(
                color: active ? Gba.onColor : Gba.ink,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _SlotRow extends StatelessWidget {
  const _SlotRow({
    required this.entry,
    required this.lane,
    required this.selected,
    required this.onTap,
  });

  final SlotEntry entry;
  final Color lane;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final pokemon = entry.pokemon;
    final onBar = selected ? Gba.onColor : Gba.ink;
    final subOnBar = selected
        ? Gba.onColor.withValues(alpha: 0.8)
        : Gba.inkSoft;

    return Semantics(
      button: true,
      selected: selected,
      label:
          'Box ${entry.box} slot ${entry.position}, ${pokemon?.speciesName ?? 'empty'}',
      child: Padding(
        padding: const EdgeInsets.fromLTRB(2, 2, 2, 2),
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            onTap: onTap,
            borderRadius: BorderRadius.circular(2),
            splashColor: lane.withValues(alpha: 0.14),
            highlightColor: lane.withValues(alpha: 0.10),
            child: AnimatedContainer(
              duration: GbaMotion.base,
              curve: GbaMotion.settle,
              decoration: BoxDecoration(
                color: selected ? lane : Colors.transparent,
                borderRadius: BorderRadius.circular(2),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 6),
              child: Row(
                children: [
                  SizedBox(
                    width: 18,
                    child: Text(
                      '${entry.position}',
                      style: GbaText.tiny.copyWith(color: subOnBar),
                    ),
                  ),
                  const SizedBox(width: 4),
                  BallBadge(ballId: pokemon?.ballId, size: 18),
                  const SizedBox(width: 6),
                  DexPlate(label: pokemon?.dexLabel ?? '---', onDark: selected, size: 30),
                  const SizedBox(width: 9),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Text(
                          pokemon == null
                              ? 'EMPTY'
                              : (pokemon.isNicknamed
                                  ? pokemon.displayName
                                  : pokemon.speciesName),
                          style: GbaText.name.copyWith(color: onBar),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          pokemon == null ? ' ' : 'OT ${pokemon.otName}',
                          style: GbaText.tiny.copyWith(color: subOnBar),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    pokemon == null ? '' : 'LV${pokemon.level}',
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
