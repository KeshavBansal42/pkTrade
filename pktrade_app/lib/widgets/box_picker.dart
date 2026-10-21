import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';
import 'gba_widgets.dart';

/// The storage screen: every box of a save, one box at a time, with the
/// Pokémon it holds. Tapping a slot hands the slot reference back so the
/// caller can trade it.
class BoxPicker extends StatefulWidget {
  const BoxPicker({super.key, required this.slots});

  final SaveSlots slots;

  /// Opens the picker and returns the chosen slot reference, or null when the
  /// player backed out.
  static Future<String?> show(BuildContext context, SaveSlots slots) {
    return showDialog<String>(
      context: context,
      builder: (dialogContext) => BoxPicker(slots: slots),
    );
  }

  @override
  State<BoxPicker> createState() => _BoxPickerState();
}

class _BoxPickerState extends State<BoxPicker> {
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

    return Dialog(
      backgroundColor: Colors.transparent,
      insetPadding: const EdgeInsets.all(14),
      child: GbaWindow(
        padding: const EdgeInsets.all(10),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Text('BOXES', style: GbaText.name),
                const Spacer(),
                Text(
                  slots.boxCount > 0 ? '$_box / ${slots.boxCount}' : 'NO BOXES',
                  style: GbaText.tiny,
                ),
              ],
            ),
            const SizedBox(height: 6),
            SizedBox(
              height: 30,
              child: ListView.builder(
                scrollDirection: Axis.horizontal,
                itemCount: numbers.length,
                itemBuilder: (context, i) {
                  final number = numbers[i];
                  final active = number == _box;
                  return Padding(
                    padding: const EdgeInsets.only(right: 4),
                    child: _BoxTab(
                      number: number,
                      active: active,
                      onTap: () => setState(() => _box = number),
                    ),
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
            Flexible(
              child: entries.isEmpty
                  ? Center(
                      child: Text(
                        'THIS BOX IS EMPTY',
                        style: GbaText.label,
                      ),
                    )
                  : ListView.builder(
                      shrinkWrap: true,
                      itemExtent: 34,
                      itemCount: entries.length + 1,
                      itemBuilder: (context, i) {
                        if (i == entries.length) {
                          final empty = slots.boxCapacity - entries.length;
                          return Padding(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 4,
                              vertical: 6,
                            ),
                            child: Text(
                              empty > 0 ? '$empty EMPTY SLOTS' : '',
                              style: GbaText.tiny.copyWith(
                                color: Gba.inkFaint,
                              ),
                            ),
                          );
                        }
                        final entry = entries[i];
                        return _SlotRow(
                          entry: entry,
                          onTap: entry.empty
                              ? null
                              : () => Navigator.of(context).pop(entry.slot),
                        );
                      },
                    ),
            ),
            const SizedBox(height: 8),
            OptionBar(
              label: 'CLOSE',
              onPressed: () => Navigator.of(context).pop(),
            ),
          ],
        ),
      ),
    );
  }
}

class _BoxTab extends StatelessWidget {
  const _BoxTab({
    required this.number,
    required this.active,
    required this.onTap,
  });

  final int number;
  final bool active;
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
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: active ? Gba.ink : Colors.transparent,
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

/// One Pokémon in a box, laid out like a party row so the two screens read
/// the same way.
class _SlotRow extends StatelessWidget {
  const _SlotRow({required this.entry, required this.onTap});

  final SlotEntry entry;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final pokemon = entry.pokemon;
    final empty = entry.empty || pokemon == null;
    final label = empty
        ? '${entry.position}'
        : '${pokemon.dexLabel} ${pokemon.displayName}';
    return Semantics(
      button: !empty,
      label: empty
          ? 'Empty slot ${entry.position}'
          : 'Box ${entry.box} slot ${entry.position}, ${pokemon.speciesName}',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(2),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
            child: Row(
              children: [
                SizedBox(
                  width: 22,
                  child: Text(
                    '${entry.position}',
                    style: GbaText.tiny.copyWith(color: Gba.inkSoft),
                  ),
                ),
                if (!empty) ...[
                  DexPlate(label: pokemon.dexLabel, size: 30),
                  const SizedBox(width: 6),
                ],
                Expanded(
                  child: Text(
                    label,
                    style: GbaText.tiny.copyWith(
                      color: empty ? Gba.inkFaint : Gba.ink,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (!empty) ...[
                  const SizedBox(width: 6),
                  Text('LV${pokemon.level}', style: GbaText.number),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
