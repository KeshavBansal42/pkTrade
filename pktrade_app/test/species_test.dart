import 'package:flutter_test/flutter_test.dart';
import 'package:pktrade_app/species.dart';

void main() {
  group('toNationalId', () {
    test('maps Generation 1 scrambled indices', () {
      expect(toNationalId(153, 1), 1); // Bulbasaur
      expect(toNationalId(176, 1), 4); // Charmander
      expect(toNationalId(1, 1), 112); // Rhydon
      expect(toNationalId(21, 1), 151); // Mew
      expect(toNationalId(190, 1), 71); // Victreebel, last valid index
    });

    test('returns null for Generation 1 MissingNo. and glitch slots', () {
      expect(toNationalId(31, 1), isNull);
      expect(toNationalId(0, 1), isNull);
      expect(toNationalId(191, 1), isNull);
      expect(toNationalId(255, 1), isNull);
    });

    test('treats Generation 2 indices as National Dex order', () {
      expect(toNationalId(25, 2), 25); // Pikachu
      expect(toNationalId(251, 2), 251); // Celebi
      expect(toNationalId(252, 2), isNull);
      expect(toNationalId(0, 2), isNull);
    });

    test('keeps the Generation 3 placeholder shift', () {
      expect(toNationalId(277, 3), 252);
      expect(toNationalId(300, 3), 275);
      expect(toNationalId(260, 3), isNull);
      expect(toNationalId(151, 3), 151);
    });
  });
}
