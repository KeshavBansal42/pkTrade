import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'models.dart';
import 'theme.dart';
import 'widgets/storage_screen.dart';
import 'widgets/transfer_preview_dialog.dart';
import 'widgets/console.dart';
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
  SaveSlots? _slotsA;
  SaveSlots? _slotsB;

  String? _selectedSlotA;
  String? _selectedSlotB;

  bool _busy = false;
  bool _picking = false;
  bool _showRoom = false;
  bool _showStorage = false;
  bool _storageForPlayerOne = true;
  int _focusedSide = 0;
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

  PartyPokemon? get _pickedA => _pickedFor(_slotsA, _selectedSlotA);
  PartyPokemon? get _pickedB => _pickedFor(_slotsB, _selectedSlotB);

  PartyPokemon? _pickedFor(SaveSlots? slots, String? ref) {
    if (slots == null || ref == null) return null;
    return slots.byRef(ref)?.pokemon;
  }

  bool get _ready =>
      _uriA != null &&
      _uriB != null &&
      _pickedA != null &&
      _pickedB != null &&
      !_busy &&
      !_picking;

  /// The side the D-pad is walking: 0 is player one, 1 is player two.
  bool get _focusIsPlayerOne => _focusedSide == 0;

  /// Moves the selection inside the focused side's party.
  void _moveSelection(int delta) {
    final data = _focusIsPlayerOne ? _saveA : _saveB;
    if (data == null || data.party.isEmpty) return;
    final current = _focusIsPlayerOne ? _selectedSlotA : _selectedSlotB;
    if (current == null) {
      // The first press lands on the top of the list.
      _selectSlot(data.party.first.slot, playerOne: _focusIsPlayerOne);
      return;
    }
    var index = 0;
    for (var i = 0; i < data.party.length; i++) {
      if (data.party[i].slot == current) {
        index = i;
        break;
      }
    }
    final next = (index + delta).clamp(0, data.party.length - 1);
    _selectSlot(data.party[next].slot, playerOne: _focusIsPlayerOne);
  }

  /// Moves the highlight to a slot on one side.
  void _selectSlot(String ref, {required bool playerOne}) {
    HapticFeedback.selectionClick();
    setState(() {
      if (playerOne) {
        _selectedSlotA = ref;
      } else {
        _selectedSlotB = ref;
      }
    });
  }

  /// The A button: pick whatever the D-pad is on, or open the storage when
  /// nothing is highlighted yet.
  void _confirmFromConsole() {
    final data = _focusIsPlayerOne ? _saveA : _saveB;
    if (data == null) return;
    final current = _focusIsPlayerOne ? _selectedSlotA : _selectedSlotB;
    if (current == null && data.party.isNotEmpty) {
      _moveSelection(0);
      return;
    }
    if (current == null) {
      _openBoxes(playerOne: _focusIsPlayerOne);
    }
  }

  void _handleDirection(DpadDirection direction) {
    switch (direction) {
      case DpadDirection.up:
        _moveSelection(-1);
      case DpadDirection.down:
        _moveSelection(1);
      case DpadDirection.left:
        setState(() => _focusedSide = 0);
      case DpadDirection.right:
        setState(() => _focusedSide = 1);
    }
  }

  /// The keyboard mirrors the console: the arrows walk the list, enter picks,
  /// escape opens the storage and the space bar sends the trade.
  KeyEventResult _handleKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.arrowUp:
        _handleDirection(DpadDirection.up);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowDown:
        _handleDirection(DpadDirection.down);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowLeft:
        _handleDirection(DpadDirection.left);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowRight:
        _handleDirection(DpadDirection.right);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.enter:
      case LogicalKeyboardKey.numpadEnter:
        _confirmFromConsole();
        return KeyEventResult.handled;
      case LogicalKeyboardKey.escape:
        _openBoxes(playerOne: _focusIsPlayerOne);
        return KeyEventResult.handled;
      case LogicalKeyboardKey.space:
        if (_ready) _openTransferPreview();
        return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

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
      final slots = await _loadSlots(uri);

      HapticFeedback.selectionClick();
      setState(() {
        if (playerOne) {
          _saveA = data;
          _slotsA = slots;
          _uriA = uri;
          _selectedSlotA = null;
        } else {
          _saveB = data;
          _slotsB = slots;
          _uriB = uri;
          _selectedSlotB = null;
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

  /// Reads the party and the boxes of a save so the storage screen can offer
  /// every Pokémon it holds.
  Future<SaveSlots?> _loadSlots(String uri) async {
    try {
      final raw = await platform.invokeMethod<String>('listSlots', {
        'uri': uri,
        'includeEmpty': true,
      });
      if (raw == null) return null;
      final decoded = jsonDecode(raw);
      if (decoded is! Map) return null;
      return SaveSlots.fromJson(Map<String, dynamic>.from(decoded));
    } on PlatformException {
      return null;
    } on FormatException {
      return null;
    }
  }

  void _pickPokemon({required bool playerOne, required int index}) {
    if (_busy) return;
    HapticFeedback.selectionClick();
    setState(() {
      final ref = 'party:$index';
      if (playerOne) {
        _selectedSlotA = _selectedSlotA == ref ? null : ref;
      } else {
        _selectedSlotB = _selectedSlotB == ref ? null : ref;
      }
    });
  }

  /// Opens the storage screen of one side so a boxed Pokémon can be picked.
  Future<void> _openBoxes({required bool playerOne}) async {
    if (_busy || _picking) return;
    final uri = playerOne ? _uriA : _uriB;
    if (uri == null) return;
    var slots = playerOne ? _slotsA : _slotsB;
    if (slots == null || !slots.hasBoxes) {
      slots = await _loadSlots(uri);
      if (slots == null) {
        _say('This save has no boxes to open.');
        return;
      }
      setState(() {
        if (playerOne) {
          _slotsA = slots;
        } else {
          _slotsB = slots;
        }
      });
    }
    if (!mounted) return;
    HapticFeedback.selectionClick();
    setState(() {
      _storageForPlayerOne = playerOne;
      _showStorage = true;
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
      final raw = await platform.invokeMethod<String>('previewSlotTrade', {
        'uriA': _uriA,
        'uriB': _uriB,
        'slotA': _selectedSlotA,
        'slotB': _selectedSlotB,
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
        builder: (dialogContext) => TransferPreviewDialog(
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
      final ok = await platform.invokeMethod('performSlotTrade', {
        'uriA': _uriA,
        'uriB': _uriB,
        'slotA': _selectedSlotA,
        'slotB': _selectedSlotB,
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
        _slotsA = null;
        _slotsB = null;
        _uriA = null;
        _uriB = null;
        _selectedSlotA = null;
        _selectedSlotB = null;
      }
    });
    if (succeeded) {
      _say('Saves updated on disk. Open them again to see the result in-game.');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Gba.shell,
      body: Focus(
        autofocus: true,
        onKeyEvent: _handleKey,
        child: DeviceShell(
          onDirection: _handleDirection,
          onConfirm: _confirmFromConsole,
          onSecondary: () => _openBoxes(playerOne: _focusIsPlayerOne),
          onStart: _ready ? _openTransferPreview : null,
          child: Stack(
            children: [
              Positioned.fill(
                child: LcdBoot(
                  child: AnimatedSwitcher(
                    duration: GbaMotion.base,
                    switchInCurve: GbaMotion.settle,
                    switchOutCurve: GbaMotion.glide,
                    transitionBuilder: (child, animation) {
                      final slide = Tween<Offset>(
                        begin: const Offset(0.18, 0),
                        end: Offset.zero,
                      ).animate(animation);
                      return FadeTransition(
                        opacity: animation,
                        child: SlideTransition(position: slide, child: child),
                      );
                    },
                    child: _showStorage ? _storage() : _table(),
                  ),
                ),
              ),
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
      ),
    );
  }

  /// The storage screen of one side, opened over the table.
  Widget _storage() {
    final playerOne = _storageForPlayerOne;
    final slots = playerOne ? _slotsA : _slotsB;
    final data = playerOne ? _saveA : _saveB;
    return Padding(
      padding: const EdgeInsets.all(2),
      child: GbaWindow(
        child: StorageScreen(
          key: ValueKey('storage-$playerOne'),
          slots: slots ??
              const SaveSlots(
                generation: 0,
                boxCount: 0,
                boxCapacity: 0,
                slots: [],
              ),
          trainerName: data?.trainerName ?? '',
          generation: slots?.generation ?? data?.generation ?? 0,
          lane: playerOne ? Gba.crimson : Gba.leaf,
          selectedSlot: playerOne ? _selectedSlotA : _selectedSlotB,
          onPick: (ref) {
            setState(() {
              if (playerOne) {
                _selectedSlotA = ref;
              } else {
                _selectedSlotB = ref;
              }
              _showStorage = false;
            });
          },
          onClose: () => setState(() => _showStorage = false),
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
              selectedSlot: _selectedSlotA,
              interactive: !_busy && !_picking,
              onSelect: (i) => _pickPokemon(playerOne: true, index: i),
              onPickFile: () => _pickFile(playerOne: true),
              onOpenBoxes: () => _openBoxes(playerOne: true),
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
              selectedSlot: _selectedSlotB,
              interactive: !_busy && !_picking,
              onSelect: (i) => _pickPokemon(playerOne: false, index: i),
              onPickFile: () => _pickFile(playerOne: false),
              onOpenBoxes: () => _openBoxes(playerOne: false),
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
