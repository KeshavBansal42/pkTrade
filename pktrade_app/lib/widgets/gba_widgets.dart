import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme.dart';

/// A 1px divider in the palette's recessed tone.
class HairRule extends StatelessWidget {
  const HairRule({super.key, this.color = Gba.shadeDeep});

  final Color color;

  @override
  Widget build(BuildContext context) => Container(height: 1, color: color);
}

/// A small bordered tag, used for the lane marker and the generation.
class TagPin extends StatelessWidget {
  const TagPin({
    super.key,
    required this.label,
    this.color = Gba.ink,
    this.fill,
  });

  final String label;
  final Color color;
  final Color? fill;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
      decoration: BoxDecoration(
        color: fill,
        border: Border.all(color: color, width: 1),
        borderRadius: BorderRadius.circular(2),
      ),
      child: Text(label, style: GbaText.tiny.copyWith(color: color)),
    );
  }
}

/// The blinking menu arrow. It answers nothing on its own, so it holds steady
/// under reduced motion and only ever marks the row or option in play.
class MenuCursor extends StatefulWidget {
  const MenuCursor({super.key, required this.color, this.size = 9});

  final Color color;
  final double size;

  @override
  State<MenuCursor> createState() => _MenuCursorState();
}

class _MenuCursorState extends State<MenuCursor>
    with SingleTickerProviderStateMixin {
  late final AnimationController _blink = AnimationController(
    vsync: this,
    duration: GbaMotion.cursorBlink,
  )..repeat(reverse: true);

  @override
  void dispose() {
    _blink.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (reducedMotion(context)) return _arrow(1);

    return AnimatedBuilder(
      animation: _blink,
      builder: (context, _) => _arrow(0.25 + _blink.value * 0.75),
    );
  }

  Widget _arrow(double t) {
    return Opacity(
      opacity: t,
      child: CustomPaint(
        size: Size(widget.size, widget.size),
        painter: _ArrowPainter(widget.color),
      ),
    );
  }
}

class _ArrowPainter extends CustomPainter {
  _ArrowPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final path = Path()
      ..moveTo(0, 0)
      ..lineTo(size.width, size.height / 2)
      ..lineTo(0, size.height)
      ..close();
    canvas.drawPath(path, Paint()..color = color);
  }

  @override
  bool shouldRepaint(_ArrowPainter old) => old.color != color;
}

/// Types a line out one character at a time, the way a handheld dialog does.
/// The caller reserves the height so the box never jumps mid-sentence.
class TypewriterText extends StatefulWidget {
  const TypewriterText({
    super.key,
    required this.text,
    required this.style,
    this.onTyped,
  });

  final String text;
  final TextStyle style;
  final VoidCallback? onTyped;

  @override
  State<TypewriterText> createState() => _TypewriterTextState();
}

class _TypewriterTextState extends State<TypewriterText>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(vsync: this)
    ..addStatusListener(_onStatus);

  int _shown = 0;
  bool _started = false;
  bool _notified = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_started) return;
    _started = true;
    // MediaQuery is only readable from here, not initState.
    _start();
  }

  @override
  void didUpdateWidget(covariant TypewriterText old) {
    super.didUpdateWidget(old);
    if (old.text != widget.text) _start();
  }

  /// Deferred a frame because the listener fires during the child's own
  /// dependency resolution, and the parent's rebuild must not be triggered
  /// mid-build.
  void _onStatus(AnimationStatus status) {
    if (status != AnimationStatus.completed || _notified) return;
    _notified = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onTyped?.call();
    });
  }

  void _start() {
    _c.stop();
    _shown = 0;
    _notified = false;
    _c.duration = Duration(
      milliseconds: (widget.text.length * GbaMotion.typeBeat.inMilliseconds)
          .clamp(120, 2400),
    );
    if (reducedMotion(context) || widget.text.isEmpty) {
      // Jumping straight to the top also drives the completion listener.
      _c.value = 1;
      return;
    }
    _c
      ..value = 0
      ..forward();
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _c,
      builder: (context, _) {
        final target = (_c.value * widget.text.length).floor();
        if (target > _shown) _shown = target;
        return Text(
          widget.text.substring(0, _shown.clamp(0, widget.text.length)),
          style: widget.style,
        );
      },
    );
  }
}

/// A full-width menu option. This is the only action colour in the interface,
/// so when it is live it is unmistakably the thing to press.
class OptionBar extends StatefulWidget {
  const OptionBar({
    super.key,
    required this.label,
    required this.onPressed,
    this.busy = false,
    this.attention = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool busy;
  final bool attention;

  @override
  State<OptionBar> createState() => _OptionBarState();
}

class _OptionBarState extends State<OptionBar> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    final enabled = widget.onPressed != null && !widget.busy;
    final fill = !enabled
        ? Gba.shade
        : _down
        ? Gba.ink
        : Gba.action;
    // Text sits on the action fill whenever the bar is live, including while
    // work is in flight.
    final inkColor = (enabled || widget.busy) ? Gba.onColor : Gba.inkFaint;
    final base = widget.busy ? Gba.action : fill;

    return Semantics(
      button: true,
      enabled: enabled,
      label: widget.label,
      child: SizedBox(
        height: 52,
        child: Stack(
          children: [
            AnimatedContainer(
              duration: GbaMotion.press,
              decoration: BoxDecoration(
                color: base,
                border: Border.all(
                  color: enabled || widget.busy ? Gba.ink : Gba.shadeDeep,
                  width: 2,
                ),
                borderRadius: BorderRadius.circular(3),
              ),
              alignment: Alignment.center,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (widget.busy)
                    const _Marquee()
                  else if (widget.attention && enabled)
                    MenuCursor(color: inkColor)
                  else
                    const SizedBox(width: 9),
                  const SizedBox(width: 9),
                  Text(
                    widget.label,
                    style: GbaText.option.copyWith(color: inkColor),
                  ),
                ],
              ),
            ),
            Positioned.fill(
              child: Material(
                color: Colors.transparent,
                child: InkWell(
                  onTap: widget.onPressed,
                  onHighlightChanged: (v) => setState(() => _down = v),
                  borderRadius: BorderRadius.circular(3),
                  splashColor: Colors.transparent,
                  highlightColor: Gba.onColor.withValues(alpha: 0.10),
                  focusColor: Gba.onColor.withValues(alpha: 0.22),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Three blocks stepping along, for work in flight.
class _Marquee extends StatefulWidget {
  const _Marquee();

  @override
  State<_Marquee> createState() => _MarqueeState();
}

class _MarqueeState extends State<_Marquee>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 720),
  )..repeat();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _c,
      builder: (context, _) {
        final step = (_c.value * 4).floor() % 4;
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (int i = 0; i < 3; i++)
              Padding(
                padding: const EdgeInsets.only(right: 3),
                child: Container(
                  width: 6,
                  height: 10,
                  color: Gba.onColor.withValues(alpha: i == step ? 1 : 0.4),
                ),
              ),
          ],
        );
      },
    );
  }
}

/// The numbered plate that stands in for a creature sprite. It carries the
/// National Dex number, which is real data the engine reports, rather than an
/// invented graphic.
class DexPlate extends StatelessWidget {
  const DexPlate({
    super.key,
    required this.label,
    this.onDark = false,
    this.size = 34,
  });

  final String label;
  final bool onDark;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: onDark ? Colors.transparent : Gba.screen,
        border: Border.all(color: onDark ? Gba.onColor : Gba.ink, width: 1),
        borderRadius: BorderRadius.circular(2),
      ),
      child: Text(
        label,
        style: GbaText.number.copyWith(color: onDark ? Gba.onColor : Gba.ink),
      ),
    );
  }
}

/// Flat cartridge mark for an unloaded save slot.
class CartridgeGlyph extends StatelessWidget {
  const CartridgeGlyph({super.key, this.color = Gba.ink, this.size = 30});

  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      size: Size(size, size),
      painter: _CartridgePainter(color),
    );
  }
}

class _CartridgePainter extends CustomPainter {
  _CartridgePainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final body = RRect.fromRectAndRadius(
      Rect.fromLTWH(w * 0.08, h * 0.06, w * 0.84, h * 0.88),
      const Radius.circular(2),
    );
    final stroke = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2;
    canvas.drawRRect(body, stroke);

    // The notch on the top edge that reads as a cartridge rather than a box.
    canvas.drawRect(
      Rect.fromLTWH(w * 0.30, h * 0.02, w * 0.24, h * 0.08),
      Paint()..color = color,
    );

    // Label band.
    canvas.drawRect(
      Rect.fromLTWH(w * 0.18, h * 0.30, w * 0.64, h * 0.16),
      Paint()..color = color,
    );

    // Contact pins.
    for (int i = 0; i < 4; i++) {
      final x = w * (0.22 + i * 0.16);
      canvas.drawLine(
        Offset(x, h * 0.62),
        Offset(x, h * 0.82),
        Paint()
          ..color = color
          ..strokeWidth = 2,
      );
    }
  }

  @override
  bool shouldRepaint(_CartridgePainter old) => old.color != color;
}

/// The indigo console body around the LCD, with the wordmark and power lamp on
/// the chin the way the hardware carries them.
class DeviceShell extends StatelessWidget {
  const DeviceShell({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: Gba.bezel,
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(10, 10, 10, 6),
          child: Column(
            children: [
              Expanded(
                child: Container(
                  decoration: BoxDecoration(
                    color: Gba.screen,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Gba.bezelDark, width: 2),
                  ),
                  padding: const EdgeInsets.all(6),
                  clipBehavior: Clip.antiAlias,
                  child: child,
                ),
              ),
              const SizedBox(height: 8),
              const _Chin(),
            ],
          ),
        ),
      ),
    );
  }
}

class _Chin extends StatelessWidget {
  const _Chin();

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        const _PowerLamp(),
        const SizedBox(width: 8),
        Text('pkTRADE', style: GbaText.wordmark),
        const Spacer(),
        Text(
          'LINK CABLE',
          style: GbaText.tiny.copyWith(
            color: Gba.onColor.withValues(alpha: 0.6),
          ),
        ),
      ],
    );
  }
}

class _PowerLamp extends StatelessWidget {
  const _PowerLamp();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 7,
      height: 7,
      decoration: BoxDecoration(
        color: Gba.crimson,
        shape: BoxShape.circle,
        border: Border.all(color: Gba.bezelLight, width: 1),
      ),
    );
  }
}

/// The single orchestrated moment in the app: the LCD drawing itself in from
/// the top when the screen wakes. Collapses to an instant paint when the
/// platform asks for reduced motion.
class LcdReveal extends StatefulWidget {
  const LcdReveal({super.key, required this.child});

  final Widget child;

  @override
  State<LcdReveal> createState() => _LcdRevealState();
}

class _LcdRevealState extends State<LcdReveal>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: GbaMotion.reveal,
  );

  bool _started = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_started) return;
    _started = true;
    // MediaQuery is only readable from here, not initState.
    if (reducedMotion(context)) {
      _c.value = 1;
    } else {
      _c.forward();
    }
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (reducedMotion(context)) return widget.child;

    return AnimatedBuilder(
      animation: _c,
      builder: (context, child) {
        final t = GbaMotion.settle.transform(_c.value);
        return LayoutBuilder(
          builder: (context, constraints) {
            final edge = constraints.maxHeight * t;
            return Stack(
              children: [
                Positioned.fill(
                  child: ClipRect(clipper: _WipeClipper(t), child: child),
                ),
                if (t < 1)
                  Positioned(
                    top: math.max(0, edge - 1),
                    left: 0,
                    right: 0,
                    child: Container(height: 2, color: Gba.inkFaint),
                  ),
              ],
            );
          },
        );
      },
      child: widget.child,
    );
  }
}

class _WipeClipper extends CustomClipper<Rect> {
  _WipeClipper(this.t);

  final double t;

  @override
  Rect getClip(Size size) => Rect.fromLTWH(0, 0, size.width, size.height * t);

  @override
  bool shouldReclip(_WipeClipper old) => old.t != t;
}
