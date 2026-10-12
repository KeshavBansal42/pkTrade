import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';
import 'gba_widgets.dart';

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
      insetPadding: const EdgeInsets.all(18),
      child: GbaWindow(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('TRANSFER CHECK', style: GbaText.name),
            const SizedBox(height: 8),
            Text(
              'PLAYER 1: ${playerOne.speciesName}\n'
              'PLAYER 2: ${playerTwo.speciesName}\n'
              'GEN $sourceGeneration → GEN $targetGeneration',
              style: GbaText.label.copyWith(color: Gba.ink),
            ),
            const SizedBox(height: 10),
            Text(
              preview.message,
              style: GbaText.label.copyWith(color: Gba.ink),
            ),
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
