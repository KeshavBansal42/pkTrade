import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';
import 'ball_art.dart';
import 'gba_widgets.dart';

/// The check the player sees before anything is written: both Pokémon, where
/// they come from, and what the engine says will happen.
class TransferPreviewDialog extends StatelessWidget {
  const TransferPreviewDialog({
    super.key,
    required this.playerOne,
    required this.playerTwo,
    required this.sourceGeneration,
    required this.targetGeneration,
    required this.preview,
    required this.onContinue,
  });

  final PartyPokemon playerOne;
  final PartyPokemon playerTwo;
  final int sourceGeneration;
  final int targetGeneration;
  final TransferPreview preview;
  final VoidCallback? onContinue;

  @override
  Widget build(BuildContext context) {
    return Dialog(
      backgroundColor: Colors.transparent,
      insetPadding: const EdgeInsets.all(14),
      child: GbaWindow(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Text('TRANSFER CHECK', style: GbaText.name),
                const Spacer(),
                Text(
                  'GEN $sourceGeneration → GEN $targetGeneration',
                  style: GbaText.tiny,
                ),
              ],
            ),
            const SizedBox(height: 10),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: _TradeCard(
                    pokemon: playerOne,
                    lane: Gba.crimson,
                    player: 'PLAYER 1',
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: _TradeCard(
                    pokemon: playerTwo,
                    lane: Gba.leaf,
                    player: 'PLAYER 2',
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              preview.message,
              style: GbaText.label.copyWith(color: Gba.ink),
            ),
            for (final evolution in preview.evolutions) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                decoration: BoxDecoration(
                  color: Gba.shade,
                  border: Border.all(color: Gba.shadeDeep),
                  borderRadius: BorderRadius.circular(2),
                ),
                child: Row(
                  children: [
                    Text('EVOLVES', style: GbaText.tiny),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        evolution.label,
                        style: GbaText.tiny.copyWith(color: Gba.ink),
                      ),
                    ),
                  ],
                ),
              ),
            ],
            if (preview.losses.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text('DATA THAT CANNOT BE KEPT', style: GbaText.label),
              const SizedBox(height: 4),
              for (final loss in preview.losses)
                Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Text('• $loss', style: GbaText.tiny),
                ),
            ],
            const SizedBox(height: 12),
            if (preview.supported)
              OptionBar(
                label: preview.lossy ? 'ACCEPT DATA LOSS' : 'CONTINUE',
                attention: true,
                onPressed: onContinue,
              )
            else ...[
              Text('NO CHANGES WILL BE MADE', style: GbaText.tiny),
              const SizedBox(height: 8),
              OptionBar(label: 'UNAVAILABLE', onPressed: null),
            ],
            const SizedBox(height: 6),
            _CancelOption(onPressed: () => Navigator.of(context).pop()),
          ],
        ),
      ),
    );
  }
}

/// One side of the trade: the creature, the ball it came in, and the slot it
/// is taken from.
class _TradeCard extends StatelessWidget {
  const _TradeCard({
    required this.pokemon,
    required this.lane,
    required this.player,
  });

  final PartyPokemon pokemon;
  final Color lane;
  final String player;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        border: Border.all(color: lane, width: 1),
        borderRadius: BorderRadius.circular(2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            color: lane,
            padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
            child: Text(
              player,
              style: GbaText.tiny.copyWith(color: Gba.onColor, fontSize: 8),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
            child: Row(
              children: [
                BallBadge(ballId: pokemon.ballId, size: 20),
                const SizedBox(width: 6),
                DexPlate(label: pokemon.dexLabel, size: 30),
                const SizedBox(width: 7),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        pokemon.speciesName,
                        style: GbaText.name.copyWith(fontSize: 12),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'LV${pokemon.level} · ${pokemon.slotLabel}',
                        style: GbaText.tiny,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _CancelOption extends StatelessWidget {
  const _CancelOption({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        foregroundColor: Gba.ink,
        minimumSize: const Size.fromHeight(44),
      ),
      child: Text('CANCEL', style: GbaText.label.copyWith(color: Gba.ink)),
    );
  }
}
