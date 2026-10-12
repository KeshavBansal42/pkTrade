import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'models.dart';
import 'theme.dart';
import 'widgets/gba_widgets.dart';
import 'widgets/link_strip.dart';
import 'widgets/trade_room.dart';
import 'widgets/trainer_window.dart';

class TradeScreen extends StatefulWidget {
  const TradeScreen({super.key});

  @override
  State<TradeScreen> createState() => _TradeScreenState();
}

class _TradeScreenState extends State<TradeScreen> {
  static const platform = MethodChannel('com.pktrade/bridge');

  String? _uriA;
  String? _uriB;
  SaveData? _saveA;
  SaveData? _saveB;

  int? _selectedA;
  int? _selectedB;

  bool _busy = false;
  bool _picking = false;
  bool _showRoom = false;
  TradeOutcome _outcome = TradeOutcome.pending;
  String? _error;
  bool _openingPreview = false;

  String? _notice;
  Timer? _noticeTimer;

  @override
  void dispose() {
    _noticeTimer?.cancel();
    super.dispose();
  }

  PartyPokemon? get _pickedA => _saveA?.byIndex(_selectedA);
  PartyPokemon? get _pickedB => _saveB?.byIndex(_selectedB);

  bool get _ready =>
      _uriA != null &&
      _uriB != null &&
      _pickedA != null &&
      _pickedB != null &&
      !_busy &&
      !_picking;

  void _say(String message) {
    _noticeTimer?.cancel();
    setState(() => _notice = message);
    _noticeTimer = Timer(const Duration(seconds: 7), () {
      if (mounted) setState(() => _notice = null);
    });
  }

  Future<void> _pickFile({required bool playerOne}) async {
    if (_picking || _busy) return;
    setState(() => _picking = true);
    try {
      final result = await platform.invokeMethod('selectSave');
      if (!mounted || result == null) return;

      final map = Map<Object?, Object?>.from(result as Map);
      final uri = map['uri'] as String?;
      final json = map['json'] as String?;
      if (uri == null || json == null) return;

      final other = playerOne ? _uriB : _uriA;
      if (other != null && other == uri) {
        _say('Both slots point at the same file. Choose a different save.');
        return;
      }

      final decoded = jsonDecode(json);
      if (decoded is! Map) {
        _say('That file could not be read as a save.');
        return;
      }
      final data = SaveData.fromJson(Map<String, dynamic>.from(decoded));

      HapticFeedback.selectionClick();
      setState(() {
        if (playerOne) {
          _saveA = data;
          _uriA = uri;
          _selectedA = null;
        } else {
          _saveB = data;
          _uriB = uri;
          _selectedB = null;
        }
      });

      if (data.isEmpty) {
        _say(
          '${data.trainerName} has an empty party, so nothing can be traded.',
        );
      }
    } on PlatformException catch (e) {
      if (mounted) _say(e.message ?? 'That save could not be opened.');
    } on FormatException {
      if (mounted) _say('That file is not valid save data.');
    } finally {
      if (mounted) setState(() => _picking = false);
    }
  }

  void _pickPokemon({required bool playerOne, required int index}) {
    if (_busy) return;
    HapticFeedback.selectionClick();
    setState(() {
      if (playerOne) {
        _selectedA = _selectedA == index ? null : index;
      } else {
        _selectedB = _selectedB == index ? null : index;
      }
    });
  }

  void _openRoom() {
    if (!_ready) return;
    HapticFeedback.mediumImpact();
    setState(() {
      _showRoom = true;
      _outcome = TradeOutcome.pending;
      _error = null;
    });
  }

  Future<void> _openTransferPreview() async {
    if (!_ready || _openingPreview) return;
    final a = _pickedA!;
    final b = _pickedB!;
    setState(() => _openingPreview = true);
    try {
      final raw = await platform.invokeMethod<String>('previewTrade', {
        'uriA': _uriA,
        'uriB': _uriB,
        'idxA': a.index,
        'idxB': b.index,
      });
      if (!mounted || raw == null) return;
      final decoded = jsonDecode(raw);
      if (decoded is! Map) {
        throw const FormatException('Invalid transfer preview');
      }
      final preview = TransferPreview.fromJson(
        Map<String, dynamic>.from(decoded),
      );
      await showDialog<void>(
        context: context,
        barrierDismissible: !preview.supported,
        builder: (dialogContext) => _TransferPreviewDialog(
          playerOne: a,
          playerTwo: b,
          sourceGeneration: _saveA!.generation,
          targetGeneration: _saveB!.generation,
          preview: preview,
          onContinue: preview.supported
              ? () {
                  Navigator.of(dialogContext).pop();
                  _openRoom();
                }
              : null,
        ),
      );
    } on PlatformException catch (e) {
      if (mounted) _say(e.message ?? 'Could not validate this transfer.');
    } on FormatException {
      if (mounted) _say('Could not read the transfer compatibility result.');
    } finally {
      if (mounted) setState(() => _openingPreview = false);
    }
  }

  Future<void> _runTrade() async {
    final a = _pickedA;
    final b = _pickedB;
    if (a == null || b == null || _uriA == null || _uriB == null) return;

    setState(() => _busy = true);
    try {
      final ok = await platform.invokeMethod('performTrade', {
        'uriA': _uriA,
        'uriB': _uriB,
        'idxA': a.index,
        'idxB': b.index,
      });
      if (!mounted) return;
      setState(() {
        if (ok == true) {
          _outcome = TradeOutcome.success;
        } else {
          _outcome = TradeOutcome.failure;
          _error = 'THE ENGINE DID NOT REPORT A COMPLETED TRADE.';
        }
      });
    } on PlatformException catch (e) {
      if (!mounted) return;
      setState(() {
        _outcome = TradeOutcome.failure;
        _error = e.message?.toUpperCase() ?? 'THE ENGINE REFUSED THE EXCHANGE.';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _closeRoom() {
    final succeeded = _outcome == TradeOutcome.success;
    setState(() {
      _showRoom = false;
      _outcome = TradeOutcome.pending;
      _error = null;
      if (succeeded) {
        // The files on disk were rewritten in place, so the previews are stale.
        _saveA = null;
        _saveB = null;
        _uriA = null;
        _uriB = null;
        _selectedA = null;
        _selectedB = null;
      }
    });
    if (succeeded) {
      _say('Saves updated on disk. Open them again to see the result in-game.');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Gba.bezel,
      body: DeviceShell(
        child: Stack(
          children: [
            Positioned.fill(child: LcdReveal(child: _table())),
            if (_showRoom)
              Positioned.fill(
                child: TradeRoom(
                  fromPlayerOne: _pickedA!,
                  fromPlayerTwo: _pickedB!,
                  outcome: _outcome,
                  errorMessage: _error,
                  onCommit: _runTrade,
                  onDismiss: _closeRoom,
                ),
              ),
            Positioned(
              top: 0,
              left: 0,
              right: 0,
              child: AnimatedSlide(
                offset: _notice == null ? const Offset(0, -1.3) : Offset.zero,
                duration: GbaMotion.base,
                curve: GbaMotion.settle,
                child: _notice == null
                    ? const SizedBox(width: double.infinity)
                    : _NoticeBox(
                        message: _notice!,
                        onDismiss: () {
                          _noticeTimer?.cancel();
                          setState(() => _notice = null);
                        },
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _table() {
    return Padding(
      padding: const EdgeInsets.all(2),
      child: Column(
        children: [
          Expanded(
            child: TrainerWindow(
              playerOne: true,
              data: _saveA,
              selectedIndex: _selectedA,
              interactive: !_busy && !_picking,
              onSelect: (i) => _pickPokemon(playerOne: true, index: i),
              onPickFile: () => _pickFile(playerOne: true),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child: LinkStrip(left: _pickedA, right: _pickedB),
          ),
          Expanded(
            child: TrainerWindow(
              playerOne: false,
              data: _saveB,
              selectedIndex: _selectedB,
              interactive: !_busy && !_picking,
              onSelect: (i) => _pickPokemon(playerOne: false, index: i),
              onPickFile: () => _pickFile(playerOne: false),
            ),
          ),
          const SizedBox(height: 8),
          OptionBar(
            label: _actionLabel,
            busy: _busy || _openingPreview,
            attention: _ready,
            onPressed: _ready && !_openingPreview ? _openTransferPreview : null,
          ),
        ],
      ),
    );
  }

  /// One label, always stating the next thing to do, so the table never needs a
  /// separate status line. Kept short enough to sit on a 320px-wide screen
  /// without wrapping.
  String get _actionLabel {
    if (_busy) return 'TRADING';
    final loaded = (_uriA != null ? 1 : 0) + (_uriB != null ? 1 : 0);
    if (loaded == 0) return 'LOAD TWO SAVES';
    if (loaded == 1) return 'LOAD OTHER SAVE';
    if (_pickedA == null && _pickedB == null) return 'PICK ONE FROM EACH';
    if (_pickedA == null) return 'PICK FROM PLAYER 1';
    if (_pickedB == null) return 'PICK FROM PLAYER 2';
    return 'TRADE';
  }
}

class _TransferPreviewDialog extends StatelessWidget {
  const _TransferPreviewDialog({
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
              'PLAYER 1: ${playerOne.speciesName}  →  GEN $targetGeneration\\n'
              'PLAYER 2: ${playerTwo.speciesName}  →  GEN $sourceGeneration',
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
              const OptionBar(label: 'UNAVAILABLE', onPressed: null),
            ],
            const SizedBox(height: 6),
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              style: TextButton.styleFrom(
                foregroundColor: Gba.ink,
                minimumSize: const Size.fromHeight(44),
              ),
              child: Text(
                'CANCEL',
                style: GbaText.label.copyWith(color: Gba.ink),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A message box that drops in from the top edge of the screen, the way the
/// handheld pushes a notice over the play field. Tapping it clears the message.
class _NoticeBox extends StatelessWidget {
  const _NoticeBox({required this.message, required this.onDismiss});

  final String message;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: 'Dismiss message',
      child: GestureDetector(
        onTap: onDismiss,
        child: GbaWindow(
          padding: const EdgeInsets.fromLTRB(10, 8, 10, 8),
          child: Row(
            children: [
              Expanded(child: Text(message, style: GbaText.label)),
              const SizedBox(width: 8),
              Text('OK', style: GbaText.tiny.copyWith(color: Gba.ink)),
            ],
          ),
        ),
      ),
    );
  }
}
