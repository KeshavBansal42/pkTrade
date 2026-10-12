import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../models.dart';
import '../theme.dart';
import 'gba_widgets.dart';

enum TradeOutcome { pending, success, failure }

/// The trade itself, played out on the LCD.
///
/// It opens on a confirm dialog the way the game asks before it writes
/// anything, closes the box while the two creatures cross, then types the
/// result. The engine's answer is what decides the outcome; nothing here
/// assumes the trade worked.
class TradeRoom extends StatefulWidget {
  const TradeRoom({
    super.key,
    required this.fromPlayerOne,
    required this.fromPlayerTwo,
    required this.outcome,
    required this.onCommit,
    required this.onDismiss,
    this.errorMessage,
  });

  final PartyPokemon fromPlayerOne;
  final PartyPokemon fromPlayerTwo;
  final TradeOutcome outcome;
  final VoidCallback onCommit;
  final VoidCallback onDismiss;
  final String? errorMessage;

  @override
  State<TradeRoom> createState() => _TradeRoomState();
}

enum _Phase { confirming, swapping, result }

class _TradeRoomState extends State<TradeRoom>
    with SingleTickerProviderStateMixin {
  late final AnimationController _swap = AnimationController(
    vsync: this,
    duration: GbaMotion.swap,
  );

  _Phase _phase = _Phase.confirming;
  bool _promptTyped = false;
  bool _dismissed = false;

  static const _homeTop = 0.25;
  static const _homeBottom = 0.75;

  @override
  void didUpdateWidget(covariant TradeRoom old) {
    super.didUpdateWidget(old);
    if (widget.outcome != old.outcome) _settle(notify: false);
  }

  void _settle({bool notify = true}) {
    if (_phase != _Phase.swapping) return;
    switch (widget.outcome) {
      case TradeOutcome.pending:
        return;
      case TradeOutcome.failure:
        _swap.stop();
        if (!reducedMotion(context)) {
          _swap.animateBack(
            0,
            duration: GbaMotion.quick,
            curve: GbaMotion.glide,
          );
        }
        _finish(notify: notify);
      case TradeOutcome.success:
        if (_swap.isCompleted) _finish(notify: notify);
    }
  }

  void _finish({bool notify = true}) {
    if (_phase == _Phase.result) return;
    HapticFeedback.mediumImpact();
    if (notify) {
      setState(() => _phase = _Phase.result);
    } else {
      _phase = _Phase.result;
    }
  }

  void _confirm() {
    HapticFeedback.mediumImpact();
    setState(() => _phase = _Phase.swapping);
    widget.onCommit();
    if (reducedMotion(context)) {
      _swap.value = 1;
      _settle();
      return;
    }
    _swap.forward().whenComplete(() {
      if (mounted) _settle();
    });
  }

  void _skip() {
    if (_phase == _Phase.swapping && !_swap.isCompleted) _swap.value = 1;
  }

  void _dismiss() {
    if (_dismissed) return;
    _dismissed = true;
    widget.onDismiss();
  }

  @override
  void dispose() {
    _swap.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _swap,
      builder: (context, _) {
        final t = _swap.value;
        // The balls rise from their trainer's side, cross vertically, then
        // settle on the opposite side. The dim wash isolates the movement.
        final crossing = _phase != _Phase.confirming;
        final dialogAway = _phase == _Phase.swapping;
        final failed = widget.outcome == TradeOutcome.failure;

        final crossU = _progress(t, 0.08, 0.62);
        final ease = Curves.easeInOut.transform(crossU);
        final topFx = crossing ? _lerp(_homeTop, _homeBottom, ease) : _homeTop;
        final bottomFx = crossing
            ? _lerp(_homeBottom, _homeTop, ease)
            : _homeBottom;
        final arc = math.sin(math.pi * crossU);

        final ready = crossing
            ? 1 + 0.04 * math.sin(math.pi * _progress(t, 0.0, 0.08))
            : 1.0;
        final landU = crossing ? _progress(t, 0.66, 0.84) : 0.0;
        final land = crossing ? 1 + 0.12 * math.sin(math.pi * landU) : 1.0;
        final flash = crossing && !failed ? _pulse(t, 0.54, 0.68) * 0.8 : 0.0;
        final dim = _phase == _Phase.swapping
            ? _lerp(0, 0.46, _progress(t, 0.0, 0.16))
            : 0.0;

        return Stack(
          children: [
            Positioned.fill(
              bottom: 170,
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: _skip,
                child: Stack(
                  clipBehavior: Clip.hardEdge,
                  children: [
                    if (dim > 0)
                      Positioned.fill(
                        child: IgnorePointer(
                          child: ColoredBox(
                            color: Gba.ink.withValues(alpha: dim),
                          ),
                        ),
                      ),
                    _ball(
                      alignmentY: topFx,
                      lift: -arc * 22,
                      scale: ready * land,
                      lane: Gba.crimson,
                      player: 'Player 1',
                      ballId: widget.fromPlayerOne.ballId,
                    ),
                    _ball(
                      alignmentY: bottomFx,
                      lift: arc * 22,
                      scale: ready * land,
                      lane: Gba.leaf,
                      player: 'Player 2',
                      ballId: widget.fromPlayerTwo.ballId,
                    ),
                    if (flash > 0)
                      Positioned.fill(
                        child: IgnorePointer(
                          child: ColoredBox(
                            color: Colors.white.withValues(alpha: flash),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ),
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: AnimatedSlide(
                offset: dialogAway ? const Offset(0, 1.15) : Offset.zero,
                duration: GbaMotion.base,
                curve: GbaMotion.settle,
                child: GbaWindow(
                  padding: const EdgeInsets.fromLTRB(10, 10, 10, 10),
                  child: _dialog(),
                ),
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _ball({
    required double alignmentY,
    required double lift,
    required double scale,
    required Color lane,
    required String player,
    required int? ballId,
  }) {
    final asset = ballSpriteAsset(ballId);
    return Align(
      alignment: Alignment(0, alignmentY * 2 - 1),
      child: Transform.translate(
        offset: Offset(0, lift),
        child: Transform.scale(
          scale: scale,
          child: Semantics(
            image: true,
            label: ballId == null
                ? '$player trade marker, ball type unknown'
                : '$player ${ballName(ballId)}',
            child: Container(
              width: 120,
              height: 120,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: Gba.window,
                shape: BoxShape.circle,
                border: Border.all(color: lane, width: 5),
              ),
              child: asset == null
                  ? const _UnknownBallMark()
                  : Image(
                      image: AssetImage(asset),
                      width: 96,
                      height: 96,
                      filterQuality: FilterQuality.none,
                      excludeFromSemantics: true,
                    ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _dialog() {
    switch (_phase) {
      case _Phase.confirming:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            TypewriterText(
              text:
                  'TRADE ${widget.fromPlayerOne.speciesName} FOR '
                  '${widget.fromPlayerTwo.speciesName}?',
              style: GbaText.prose,
              onTyped: () {
                if (mounted && !_promptTyped) {
                  setState(() => _promptTyped = true);
                }
              },
            ),
            const SizedBox(height: 10),
            if (_promptTyped) ...[
              OptionBar(label: 'YES', attention: true, onPressed: _confirm),
              const SizedBox(height: 8),
              _PlainOption(label: 'NO', onPressed: _dismiss),
            ],
          ],
        );

      case _Phase.swapping:
        return const SizedBox.shrink();

      case _Phase.result:
        final failed = widget.outcome == TradeOutcome.failure;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            TypewriterText(
              text: failed ? 'TRADE FAILED' : 'TRADE COMPLETE',
              style: GbaText.prose.copyWith(
                fontWeight: FontWeight.w700,
                color: Gba.ink,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              failed
                  ? (widget.errorMessage ?? 'THE ENGINE REFUSED THE EXCHANGE.')
                  : '${widget.fromPlayerOne.speciesName} NOW BELONGS TO '
                        'PLAYER 2.\n${widget.fromPlayerTwo.speciesName} NOW '
                        'BELONGS TO PLAYER 1.',
              style: GbaText.label,
            ),
            const SizedBox(height: 10),
            OptionBar(label: 'OK', attention: true, onPressed: _dismiss),
          ],
        );
    }
  }
}

/// A secondary menu choice: plain text with a rule, the way a handheld lists a
/// non-highlighted option.
class _PlainOption extends StatelessWidget {
  const _PlainOption({required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: label,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onPressed,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 2),
            child: Row(
              children: [
                const SizedBox(width: 18),
                Text(label, style: GbaText.label.copyWith(color: Gba.ink)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String? ballSpriteAsset(int? id) {
  const names = <int, String>{
    1: 'master-ball',
    2: 'ultra-ball',
    3: 'great-ball',
    4: 'poke-ball',
    5: 'safari-ball',
    6: 'net-ball',
    7: 'dive-ball',
    8: 'nest-ball',
    9: 'repeat-ball',
    10: 'timer-ball',
    11: 'luxury-ball',
    12: 'premier-ball',
  };
  final name = id == null ? null : names[id];
  return name == null ? null : 'assets/images/balls/$name.png';
}

String ballName(int id) {
  const names = <int, String>{
    1: 'MASTER BALL',
    2: 'ULTRA BALL',
    3: 'GREAT BALL',
    4: 'POKÉ BALL',
    5: 'SAFARI BALL',
    6: 'NET BALL',
    7: 'DIVE BALL',
    8: 'NEST BALL',
    9: 'REPEAT BALL',
    10: 'TIMER BALL',
    11: 'LUXURY BALL',
    12: 'PREMIER BALL',
  };
  return names[id] ?? 'UNKNOWN BALL';
}

class _UnknownBallMark extends StatelessWidget {
  const _UnknownBallMark();

  @override
  Widget build(BuildContext context) => Column(
    mainAxisSize: MainAxisSize.min,
    children: [
      const CartridgeGlyph(color: Gba.inkSoft, size: 32),
      const SizedBox(height: 5),
      Text('UNKNOWN', style: GbaText.tiny),
    ],
  );
}

double _progress(double t, double begin, double end) =>
    ((t - begin) / (end - begin)).clamp(0.0, 1.0);

double _pulse(double t, double begin, double end) {
  if (t <= begin || t >= end) return 0;
  return math.sin(math.pi * (t - begin) / (end - begin));
}

double _lerp(double a, double b, double u) => a + (b - a) * u;
