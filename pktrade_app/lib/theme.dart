import 'package:flutter/material.dart';

/// Colour, type and motion for the handheld UI.
///
/// The machine is a handheld console, not a terminal: a plum plastic shell
/// with a moulded lip, a dark bezel sunk into it, and a bright LCD in the
/// middle. The screen is a reflective panel, not a dark app: a flat cream
/// field and navy ink. Depth comes from borders and two-tone bevels, never from
/// gradients or soft shadows on the screen itself; the only gradients live on
/// the plastic around it.
class Gba {
  Gba._();

  // Console shell: plum plastic with a lighter moulded lip.
  static const shell = Color(0xFF3A2F5E);
  static const shellLight = Color(0xFF544479);
  static const shellDark = Color(0xFF241C40);
  static const shellEdge = Color(0xFF6A5896);

  // Screen bezel, sunk into the shell.
  static const bezel = Color(0xFF171226);
  static const bezelLight = Color(0xFF241B3C);

  // Console controls.
  static const button = Color(0xFF4C3E75);
  static const buttonLight = Color(0xFF6A5896);
  static const buttonDark = Color(0xFF2A2148);
  static const lamp = Color(0xFF6BE86B);
  static const grille = Color(0xFF1B1430);

  // LCD surface
  static const screen = Color(0xFFF2F2E9);
  static const window = Color(0xFFFBFBF5);
  static const shade = Color(0xFFE3E3D5);
  static const shadeDeep = Color(0xFFCBCBBC);

  // Ink
  static const ink = Color(0xFF12204A);
  static const inkSoft = Color(0xFF5C6C96);
  static const inkFaint = Color(0xFF97A2C0);

  // Roles. The two version colours identify the two saves; blue is the one
  // action colour in the whole interface.
  static const crimson = Color(0xFFBE2A22);
  static const leaf = Color(0xFF1C7A3B);
  static const action = Color(0xFF2C5FD8);
  static const onColor = Color(0xFFF7F7EE);
}

/// Silkscreen for everything. A real handheld ships one bitmap face, so the
/// hierarchy comes from size and weight rather than a second family, and every
/// string stays uppercase the way the platform's own font did.
class GbaText {
  GbaText._();

  static const _face = 'Silkscreen';

  static const wordmark = TextStyle(
    fontFamily: _face,
    fontSize: 13,
    fontWeight: FontWeight.w700,
    height: 1.2,
    color: Gba.onColor,
  );

  static const windowTitle = TextStyle(
    fontFamily: _face,
    fontSize: 11,
    fontWeight: FontWeight.w700,
    height: 1.3,
    color: Gba.onColor,
  );

  static const name = TextStyle(
    fontFamily: _face,
    fontSize: 14,
    fontWeight: FontWeight.w700,
    height: 1.3,
    color: Gba.ink,
  );

  static const label = TextStyle(
    fontFamily: _face,
    fontSize: 10,
    height: 1.5,
    color: Gba.inkSoft,
  );

  static const tiny = TextStyle(
    fontFamily: _face,
    fontSize: 9,
    height: 1.5,
    color: Gba.inkSoft,
  );

  static const data = TextStyle(
    fontFamily: _face,
    fontSize: 12,
    height: 1.3,
    color: Gba.ink,
  );

  static const number = TextStyle(
    fontFamily: _face,
    fontSize: 12,
    fontWeight: FontWeight.w700,
    height: 1.2,
    color: Gba.ink,
  );

  /// Dialog and body copy. Roomier line-height because it is read as prose.
  static const prose = TextStyle(
    fontFamily: _face,
    fontSize: 14,
    height: 1.7,
    color: Gba.ink,
  );

  static const option = TextStyle(
    fontFamily: _face,
    fontSize: 13,
    fontWeight: FontWeight.w700,
    height: 1.3,
    color: Gba.onColor,
  );
}

/// Motion is mostly a response to input. The only unrequested movement is the
/// single LCD power-on reveal; everything else answers a tap.
class GbaMotion {
  GbaMotion._();

  static const press = Duration(milliseconds: 80);
  static const quick = Duration(milliseconds: 140);
  static const base = Duration(milliseconds: 220);
  static const reveal = Duration(milliseconds: 420);
  static const typeBeat = Duration(milliseconds: 26);
  static const swap = Duration(milliseconds: 2200);
  static const cursorBlink = Duration(milliseconds: 560);
  static const boot = Duration(milliseconds: 900);

  static const settle = Cubic(0.2, 1, 0.3, 1);
  static const glide = Cubic(0.4, 0, 0.2, 1);
  static const snap = Cubic(0.85, 0, 0.15, 1);
  static const power = Cubic(0.1, 0.9, 0.2, 1);
}

/// True when the platform asks for reduced motion. Long reveals collapse to an
/// instant state change, and blinking cursors hold steady.
bool reducedMotion(BuildContext context) =>
    MediaQuery.disableAnimationsOf(context);

/// Every window shares the same three-layer frame: a navy outer rule, a single
/// pixel of pale shade, then the white interior. That stacking is what reads as
/// a handheld dialog box rather than a card.
class GbaWindow extends StatelessWidget {
  const GbaWindow({
    super.key,
    required this.child,
    this.padding = EdgeInsets.zero,
    this.radius = 4,
    this.fill = Gba.window,
    this.borderColor = Gba.ink,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final double radius;
  final Color fill;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: borderColor,
        borderRadius: BorderRadius.circular(radius),
      ),
      padding: const EdgeInsets.all(2),
      child: Container(
        decoration: BoxDecoration(
          color: Gba.shade,
          borderRadius: BorderRadius.circular((radius - 1).clamp(0, radius)),
        ),
        padding: const EdgeInsets.all(1),
        child: Container(
          decoration: BoxDecoration(
            color: fill,
            borderRadius: BorderRadius.circular((radius - 2).clamp(0, radius)),
          ),
          padding: padding,
          clipBehavior: Clip.antiAlias,
          child: child,
        ),
      ),
    );
  }
}
