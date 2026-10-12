import 'package:flutter/material.dart';

import '../models.dart';
import '../theme.dart';

/// The cable between the two trainers. It shows the pending exchange before
/// anything is written: two outline bays while empty, and two filled chips with
/// an energised line once both sides have committed.
class LinkStrip extends StatefulWidget {
  const LinkStrip({super.key, required this.left, required this.right});

  final PartyPokemon? left;
  final PartyPokemon? right;

  @override
  State<LinkStrip> createState() => _LinkStripState();
}

class _LinkStripState extends State<LinkStrip>
    with SingleTickerProviderStateMixin {
  late final AnimationController _flow = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 900),
  );

  bool get _live => widget.left != null && widget.right != null;

  @override
  void initState() {
    super.initState();
    _sync();
  }

  @override
  void didUpdateWidget(covariant LinkStrip old) {
    super.didUpdateWidget(old);
    if (old.left != widget.left || old.right != widget.right) _sync();
  }

  void _sync() {
    if (_live && !_flow.isAnimating) {
      _flow.repeat();
    } else if (!_live && _flow.isAnimating) {
      _flow.stop();
      _flow.value = 0;
    }
  }

  @override
  void dispose() {
    _flow.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final reduced = reducedMotion(context);
    return SizedBox(
      height: 44,
      child: Row(
        children: [
          _Bay(pokemon: widget.left, lane: Gba.crimson),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 6),
              child: AnimatedBuilder(
                animation: _flow,
                builder: (context, _) => CustomPaint(
                  size: Size.infinite,
                  painter: _RailPainter(
                    phase: reduced ? 0.5 : _flow.value,
                    live: _live,
                  ),
                ),
              ),
            ),
          ),
          _Bay(pokemon: widget.right, lane: Gba.leaf),
        ],
      ),
    );
  }
}

class _Bay extends StatelessWidget {
  const _Bay({required this.pokemon, required this.lane});

  final PartyPokemon? pokemon;
  final Color lane;

  @override
  Widget build(BuildContext context) {
    final p = pokemon;
    final chosen = p != null;
    return Container(
      constraints: const BoxConstraints(maxWidth: 124),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
      decoration: BoxDecoration(
        color: chosen ? lane : Colors.transparent,
        border: Border.all(color: chosen ? lane : Gba.inkFaint, width: 1),
        borderRadius: BorderRadius.circular(2),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (chosen) ...[
            Text(
              p.dexLabel,
              style: GbaText.number.copyWith(color: Gba.onColor),
            ),
            const SizedBox(width: 6),
          ],
          Flexible(
            child: Text(
              chosen ? p.speciesName : 'EMPTY',
              style: GbaText.tiny.copyWith(
                color: chosen ? Gba.onColor : Gba.inkFaint,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

class _RailPainter extends CustomPainter {
  _RailPainter({required this.phase, required this.live});

  final double phase;
  final bool live;

  @override
  void paint(Canvas canvas, Size size) {
    final y = size.height / 2;
    final base = Paint()
      ..color = Gba.inkFaint
      ..strokeWidth = 2;
    canvas.drawLine(Offset(0, y), Offset(size.width, y), base);

    if (live) {
      const dash = 7.0;
      const gap = 5.0;
      const period = dash + gap;
      double x = -period + phase * period;
      while (x < size.width) {
        final start = x.clamp(0.0, size.width);
        final end = (x + dash).clamp(0.0, size.width);
        if (end > start) {
          final t = ((start + end) / 2) / size.width;
          canvas.drawLine(
            Offset(start, y),
            Offset(end, y),
            Paint()
              ..color = Color.lerp(Gba.crimson, Gba.leaf, t)!
              ..strokeWidth = 3,
          );
        }
        x += period;
      }
    }

    // Plugs at each end of the cable.
    final plug = Paint()..color = Gba.ink;
    canvas.drawRect(
      Rect.fromCenter(center: Offset(1, y), width: 3, height: 14),
      plug,
    );
    canvas.drawRect(
      Rect.fromCenter(center: Offset(size.width - 1, y), width: 3, height: 14),
      plug,
    );
  }

  @override
  bool shouldRepaint(_RailPainter old) =>
      old.phase != phase || old.live != live;
}
