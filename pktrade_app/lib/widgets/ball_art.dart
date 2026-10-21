import 'package:flutter/material.dart';

import '../theme.dart';

/// The capture ball artwork, keyed by the item number the games use.
///
/// The twelve classic balls are 1 to 12 in every generation. The four balls
/// that arrived with Gen 4 are 13 to 16, the apricorn balls are 492 to 500,
/// the Dream Ball is 576 and the Beast Ball is 851.
const Map<int, String> ballAssets = <int, String>{
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
  13: 'dusk-ball',
  14: 'heal-ball',
  15: 'quick-ball',
  16: 'cherish-ball',
  492: 'fast-ball',
  493: 'level-ball',
  494: 'lure-ball',
  495: 'heavy-ball',
  496: 'love-ball',
  497: 'friend-ball',
  498: 'moon-ball',
  499: 'sport-ball',
  500: 'park-ball',
  576: 'dream-ball',
  851: 'beast-ball',
};

const Map<int, String> ballNames = <int, String>{
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
  13: 'DUSK BALL',
  14: 'HEAL BALL',
  15: 'QUICK BALL',
  16: 'CHERISH BALL',
  492: 'FAST BALL',
  493: 'LEVEL BALL',
  494: 'LURE BALL',
  495: 'HEAVY BALL',
  496: 'LOVE BALL',
  497: 'FRIEND BALL',
  498: 'MOON BALL',
  499: 'SPORT BALL',
  500: 'PARK BALL',
  576: 'DREAM BALL',
  851: 'BEAST BALL',
};

/// The sprite for a capture ball, or null when the ball is unknown.
String? ballSpriteAsset(int? id) {
  final name = id == null ? null : ballAssets[id];
  return name == null ? null : 'assets/images/balls/$name.png';
}

/// The name of a capture ball.
String ballName(int id) => ballNames[id] ?? 'UNKNOWN BALL';

/// A capture ball drawn the way the game draws it: the sprite, framed by a
/// hairline so it reads as an object rather than a picture.
class BallBadge extends StatelessWidget {
  const BallBadge({super.key, required this.ballId, this.size = 18});

  final int? ballId;
  final double size;

  @override
  Widget build(BuildContext context) {
    final asset = ballSpriteAsset(ballId);
    if (asset == null) {
      return SizedBox(
        width: size,
        height: size,
        child: Center(
          child: Text(
            '?',
            style: GbaText.tiny.copyWith(color: Gba.inkFaint),
          ),
        ),
      );
    }
    return Image(
      image: AssetImage(asset),
      width: size,
      height: size,
      fit: BoxFit.contain,
      filterQuality: FilterQuality.none,
      excludeFromSemantics: true,
    );
  }
}
