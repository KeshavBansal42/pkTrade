import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme.dart';

/// The directions the D-pad can report.
enum DpadDirection { up, down, left, right }

/// The console the app lives in: a plum handheld with a screen sunk into it
/// and real controls along the chin.
///
/// The controls are not decoration. The D-pad walks the party list, A picks
/// the highlighted Pokémon, B opens the storage and START sends the trade, so
/// the machine can be driven with a thumb or with the keyboard.
class DeviceShell extends StatelessWidget {
  const DeviceShell({
    super.key,
    required this.child,
    this.onDirection,
    this.onConfirm,
    this.onSecondary,
    this.onStart,
    this.startEnabled = false,
  });

  final Widget child;
  final ValueChanged<DpadDirection>? onDirection;
  final VoidCallback? onConfirm;
  final VoidCallback? onSecondary;
  final VoidCallback? onStart;
  final bool startEnabled;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Gba.shellLight, Gba.shell, Gba.shellDark],
          stops: [0.0, 0.45, 1.0],
        ),
      ),
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(8, 8, 8, 4),
          child: Column(
            children: [
              Expanded(child: _screen()),
              const SizedBox(height: 6),
              _chin(),
            ],
          ),
        ),
      ),
    );
  }

  /// The bezel with the LCD dropped into it.
  Widget _screen() {
    return Container(
      decoration: BoxDecoration(
        color: Gba.bezel,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Gba.shellEdge.withValues(alpha: 0.5), width: 1),
        boxShadow: const [
          BoxShadow(
            color: Color(0x66000000),
            blurRadius: 12,
            offset: Offset(0, 4),
          ),
        ],
      ),
      padding: const EdgeInsets.all(6),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(5),
        child: Stack(
          children: [
            Positioned.fill(child: ColoredBox(color: Gba.screen)),
            Positioned.fill(child: child),
            // The panel glass: a faint scan pattern and a corner vignette.
            const Positioned.fill(child: IgnorePointer(child: _Scanlines())),
          ],
        ),
      ),
    );
  }

  Widget _chin() {
    return Container(
      height: 76,
      decoration: BoxDecoration(
        color: Gba.shellDark.withValues(alpha: 0.55),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Gba.shellEdge.withValues(alpha: 0.22)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      child: LayoutBuilder(
        builder: (context, constraints) {
          // On a narrow phone the speaker and the wide pills give way so the
          // controls always fit.
          final compact = constraints.maxWidth < 360;
          return Row(
        children: [
          ConsoleDpad(
            onDirection: onDirection,
            size: math.min(26.0, constraints.maxHeight / 3.1),
          ),
          const SizedBox(width: 8),
          if (!compact) ...[
            const SpeakerGrille(),
            const SizedBox(width: 10),
          ],
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    ConsolePill(
                      label: 'SELECT',
                      onPressed: onSecondary,
                      compact: compact,
                    ),
                    const SizedBox(width: 6),
                    ConsolePill(
                      label: 'START',
                      onPressed: onStart,
                      compact: compact,
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const PowerLamp(),
                    const SizedBox(width: 6),
                    Text(
                      'pkTRADE',
                      style: GbaText.wordmark.copyWith(
                        color: Gba.onColor.withValues(alpha: 0.85),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Row(
            children: [
              ConsoleButton(
                label: 'B',
                onPressed: onSecondary,
                scale: math.min(1.0, constraints.maxHeight / 40),
              ),
              const SizedBox(width: 6),
              ConsoleButton(
                label: 'A',
                primary: true,
                onPressed: onConfirm,
                scale: math.min(1.0, constraints.maxHeight / 40),
              ),
            ],
          ),
        ],
      );
        },
      ),
    );
  }
}

/// The cross of four directions, pressed like the real thing.
class ConsoleDpad extends StatelessWidget {
  const ConsoleDpad({super.key, this.onDirection, this.size = 30.0});

  final ValueChanged<DpadDirection>? onDirection;
  final double size;

  @override
  Widget build(BuildContext context) {
    Widget pad(DpadDirection direction, Widget child) {
      return _PadKey(
        direction: direction,
        size: size,
        onPressed: onDirection == null ? null : () => onDirection!(direction),
        child: child,
      );
    }

    return SizedBox(
      width: size * 3,
      height: size * 3,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Padding(
            padding: EdgeInsets.only(top: size),
            child: pad(DpadDirection.down, const SizedBox.shrink()),
          ),
          Padding(
            padding: EdgeInsets.only(bottom: size),
            child: pad(DpadDirection.up, const SizedBox.shrink()),
          ),
          Padding(
            padding: EdgeInsets.only(right: size),
            child: pad(DpadDirection.left, const SizedBox.shrink()),
          ),
          Padding(
            padding: EdgeInsets.only(left: size),
            child: pad(DpadDirection.right, const SizedBox.shrink()),
          ),
        ],
      ),
    );
  }
}

class _PadKey extends StatefulWidget {
  const _PadKey({
    required this.direction,
    required this.size,
    required this.onPressed,
    required this.child,
  });

  final DpadDirection direction;
  final double size;
  final VoidCallback? onPressed;
  final Widget child;

  @override
  State<_PadKey> createState() => _PadKeyState();
}

class _PadKeyState extends State<_PadKey> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      enabled: widget.onPressed != null,
      label: 'D-pad ${widget.direction.name}',
      child: GestureDetector(
        onTapDown: (_) => setState(() => _down = true),
        onTapUp: (_) => setState(() => _down = false),
        onTapCancel: () => setState(() => _down = false),
        onTap: widget.onPressed,
        child: AnimatedContainer(
          duration: GbaMotion.press,
          curve: GbaMotion.settle,
          margin: EdgeInsets.zero,
          decoration: BoxDecoration(
            color: _down ? Gba.buttonDark : Gba.buttonLight,
            borderRadius: BorderRadius.circular(widget.size * 0.28),
            border: Border.all(color: Gba.shellEdge.withValues(alpha: 0.35)),
            boxShadow: _down
                ? null
                : const [
                    BoxShadow(
                      color: Color(0xAA000000),
                      blurRadius: 3,
                      offset: Offset(0, 2),
                    ),
                  ],
          ),
          child: widget.child,
        ),
      ),
    );
  }
}

/// A round console button. The primary one is the action colour.
class ConsoleButton extends StatefulWidget {
  const ConsoleButton({
    super.key,
    required this.label,
    this.onPressed,
    this.primary = false,
    this.scale = 1.0,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool primary;
  final double scale;

  @override
  State<ConsoleButton> createState() => _ConsoleButtonState();
}

class _ConsoleButtonState extends State<ConsoleButton> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    final enabled = widget.onPressed != null;
    final face = !enabled
        ? Gba.buttonDark
        : widget.primary
        ? Gba.action
        : Gba.button;
    return Semantics(
      button: true,
      enabled: enabled,
      label: '${widget.label} button',
      child: GestureDetector(
        onTapDown: enabled ? (_) => setState(() => _down = true) : null,
        onTapUp: enabled ? (_) => setState(() => _down = false) : null,
        onTapCancel: enabled ? () => setState(() => _down = false) : null,
        onTap: widget.onPressed,
        child: AnimatedContainer(
          duration: GbaMotion.press,
          curve: GbaMotion.settle,
          width: 34 * widget.scale,
          height: 34 * widget.scale,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: _down ? Gba.buttonDark : face,
            shape: BoxShape.circle,
            border: Border.all(color: Gba.shellEdge.withValues(alpha: 0.45)),
            boxShadow: _down
                ? null
                : const [
                    BoxShadow(
                      color: Color(0x99000000),
                      blurRadius: 4,
                      offset: Offset(0, 2),
                    ),
                  ],
          ),
          child: Text(
            widget.label,
            style: GbaText.option.copyWith(
              color: enabled ? Gba.onColor : Gba.onColor.withValues(alpha: 0.4),
              fontSize: 12,
            ),
          ),
        ),
      ),
    );
  }
}

/// The small rubber pills marked START and SELECT.
class ConsolePill extends StatelessWidget {
  const ConsolePill({
    super.key,
    required this.label,
    this.onPressed,
    this.compact = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final enabled = onPressed != null;
    return Semantics(
      button: true,
      enabled: enabled,
      label: label,
      child: GestureDetector(
        onTap: onPressed,
        child: Container(
          padding: EdgeInsets.symmetric(
            horizontal: compact ? 5 : 8,
            vertical: compact ? 2 : 3,
          ),
          decoration: BoxDecoration(
            color: enabled ? Gba.button : Gba.buttonDark,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: Gba.shellEdge.withValues(alpha: 0.35)),
          ),
          child: Text(
            label,
            style: GbaText.tiny.copyWith(
              color: enabled ? Gba.onColor : Gba.onColor.withValues(alpha: 0.4),
              fontSize: 8,
            ),
          ),
        ),
      ),
    );
  }
}

/// The power lamp, lit while the machine is on.
class PowerLamp extends StatelessWidget {
  const PowerLamp({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 7,
      height: 7,
      decoration: BoxDecoration(
        color: Gba.lamp,
        shape: BoxShape.circle,
        boxShadow: const [
          BoxShadow(color: Gba.lamp, blurRadius: 6, spreadRadius: 0.5),
        ],
      ),
    );
  }
}

/// The speaker holes on the side of the chin.
class SpeakerGrille extends StatelessWidget {
  const SpeakerGrille({super.key, this.columns = 5, this.rows = 3});

  final int columns;
  final int rows;

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      size: Size(columns * 6.0, rows * 6.0),
      painter: _GrillePainter(),
    );
  }
}

class _GrillePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = Gba.grille;
    const step = 6.0;
    for (var y = step / 2; y < size.height; y += step) {
      for (var x = step / 2; x < size.width; x += step) {
        canvas.drawCircle(Offset(x, y), 1.6, paint);
      }
    }
  }

  @override
  bool shouldRepaint(_GrillePainter old) => false;
}

/// The glass over the panel: a faint scan pattern that keeps the cream field
/// reading as a screen rather than paper.
class _Scanlines extends StatelessWidget {
  const _Scanlines();

  @override
  Widget build(BuildContext context) {
    return CustomPaint(painter: _ScanlinePainter());
  }
}

class _ScanlinePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final line = Paint()..color = const Color(0x0A12204A);
    for (var y = 0.0; y < size.height; y += 3) {
      canvas.drawRect(Rect.fromLTWH(0, y, size.width, 1), line);
    }

    // A soft vignette in the corners, like the glass catching the light.
    final vignette = Paint()
      ..shader = RadialGradient(
        radius: 0.95,
        colors: const [Color(0x00000000), Color(0x1412204A)],
        stops: const [0.65, 1.0],
      ).createShader(Offset.zero & size);
    canvas.drawRect(Offset.zero & size, vignette);
  }

  @override
  bool shouldRepaint(_ScanlinePainter old) => false;
}

/// The power-on sweep: the panel lights up from the middle and the picture
/// settles into place. Collapses to an instant paint under reduced motion.
class LcdBoot extends StatefulWidget {
  const LcdBoot({super.key, required this.child});

  final Widget child;

  @override
  State<LcdBoot> createState() => _LcdBootState();
}

class _LcdBootState extends State<LcdBoot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: GbaMotion.boot,
  );

  bool _started = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_started) return;
    _started = true;
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
        final t = GbaMotion.power.transform(_c.value);
        return Stack(
          fit: StackFit.expand,
          children: [
            ?child,
            // The panel brightening as it wakes.
            IgnorePointer(
              child: ColoredBox(
                color: Gba.onColor.withValues(alpha: (1 - t) * 0.9),
              ),
            ),
            // A line of light sweeping down the panel.
            IgnorePointer(
              child: Align(
                alignment: Alignment(0, -1 + 2 * t),
                child: FractionallySizedBox(
                  widthFactor: 1,
                  heightFactor: 0.06,
                  child: ColoredBox(
                    color: Gba.onColor.withValues(alpha: (1 - t) * 0.5),
                  ),
                ),
              ),
            ),
          ],
        );
      },
      child: widget.child,
    );
  }
}

/// A soft glow used behind the screen while a trade runs.
class ScreenGlow extends StatelessWidget {
  const ScreenGlow({super.key, required this.color, required this.strength});

  final Color color;
  final double strength;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: DecoratedBox(
        decoration: BoxDecoration(
          gradient: RadialGradient(
            radius: 0.8,
            colors: [color.withValues(alpha: 0.35 * strength), color.withValues(alpha: 0)],
          ),
        ),
      ),
    );
  }
}

/// Keeps a square inside a box, used where a sprite has to stay round.
class SquareBox extends StatelessWidget {
  const SquareBox({super.key, required this.child, this.size});

  final Widget child;
  final double? size;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: AspectRatio(aspectRatio: 1, child: child),
    );
  }
}

/// The math the shell and the room share.
double clampUnit(double value) => math.max(0, math.min(1, value));
