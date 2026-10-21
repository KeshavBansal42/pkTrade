import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pktrade_app/theme.dart';
import 'package:pktrade_app/trade_screen.dart';
import 'package:pktrade_app/widgets/gba_widgets.dart';
import 'package:pktrade_app/widgets/trade_room.dart';
import 'package:pktrade_app/widgets/trainer_window.dart';

const _channel = MethodChannel('com.pktrade/bridge');

const _saveA =
    '{"generation":3,"trainerName":"UniversalTrainer","tid":0,"sid":0,"party":['
    '{"index":1,"species":64,"level":16,"ballId":1,"nickname":"KADABRA","otName":"AAAAA","otId":33403},'
    '{"index":2,"species":93,"level":22,"nickname":"SPOOKY","otName":"AAAAA","otId":33403}]}';

const _saveB =
    '{"generation":3,"trainerName":"UniversalTrainer","tid":0,"sid":0,"party":['
    '{"index":1,"species":67,"level":18,"ballId":12,"nickname":"MACHOKE","otName":"EE","otId":22615},'
    '{"index":2,"species":75,"level":24,"nickname":"ROCKY","otName":"EE","otId":22615}]}';

Future<void> _loadFonts() async {
  final loader = FontLoader('Silkscreen')
    ..addFont(rootBundle.load('assets/fonts/Silkscreen-Regular.ttf'))
    ..addFont(rootBundle.load('assets/fonts/Silkscreen-Bold.ttf'));
  await loader.load();
}

/// Builds the slot listing the engine would return for a save: the party
/// plus one box with a single occupant.
Map<String, dynamic> slotsPayload(String? partyJson) {
  final decoded = partyJson == null
      ? <String, dynamic>{}
      : Map<String, dynamic>.from(jsonDecode(partyJson) as Map);
  final party = (decoded['party'] as List?) ?? const [];
  final slots = <Map<String, dynamic>>[];
  for (final entry in party) {
    final slot = Map<String, dynamic>.from(entry as Map);
    slot['slot'] = 'party:${slot['index']}';
    slot['empty'] = false;
    slots.add(slot);
  }
  slots.add({
    'slot': 'box:1/1',
    'empty': false,
    'index': 1,
    'species': 25,
    'level': 12,
    'nickname': 'PIKACHU',
    'otName': 'AAAAA',
    'otId': 33403,
    'ballId': 4,
  });
  slots.add({'slot': 'box:1/2', 'empty': true});
  return {
    'generation': decoded['generation'] ?? 3,
    'boxCount': 14,
    'boxCapacity': 30,
    'party': party,
    'slots': slots,
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late List<String> pending;
  late Map<String, String> selected;
  late Object tradeResult; // true, false, or 'throw'
  late int tradeCalls;
  late bool previewSupported;
  late List<String> previewLosses;

  setUp(() {
    pending = [_saveA, _saveB];
    selected = {};
    tradeResult = true;
    tradeCalls = 0;
    previewSupported = true;
    previewLosses = [];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, (call) async {
          switch (call.method) {
            case 'selectSave':
              if (pending.isEmpty) return null;
              final uri = 'content://s/${pending.length}';
              final json = pending.removeAt(0);
              selected[uri] = json;
              return {'uri': uri, 'json': json};
            case 'listSlots':
              final uri = (call.arguments as Map)['uri'] as String;
              return jsonEncode(slotsPayload(selected[uri]));
            case 'previewTrade':
            case 'previewSlotTrade':
              return jsonEncode({
                'supported': previewSupported,
                'lossy': previewLosses.isNotEmpty,
                'message': previewSupported
                    ? 'Verified Gen 3 record swap'
                    : 'Verified party writer unavailable',
                'losses': previewLosses,
              });
            case 'performTrade':
            case 'performSlotTrade':
              tradeCalls++;
              if (tradeResult == 'throw') {
                throw PlatformException(
                  code: 'TRADE_ERROR',
                  message: 'checksum mismatch',
                );
              }
              return tradeResult;
          }
          return null;
        });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, null);
  });

  /// Drives the table up to the point where both sides have a Pokemon chosen.
  Future<void> setUpTable(
    WidgetTester tester, {
    Size size = const Size(390, 844),
  }) async {
    await _loadFonts();
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 600));
    await tester.tap(find.text('KADABRA').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 200));
    await tester.tap(find.text('MACHOKE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 600));
  }

  /// Opens the room and waits for the prompt to finish typing.
  Future<void> openRoom(WidgetTester tester) async {
    await tester.tap(find.text('TRADE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump();
    expect(find.text('TRANSFER CHECK'), findsOneWidget);
    await tester.tap(find.text('CONTINUE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 1200));
    await tester.pump();
  }

  /// Runs the swap out and lets the result dialog type itself.
  Future<void> runResult(WidgetTester tester) async {
    await tester.pump(); // establish the swap ticker
    await tester.pump(const Duration(milliseconds: 2400)); // the swap lands
    await tester.pump(); // the result dialog mounts
    await tester.pump(const Duration(milliseconds: 700)); // result types out
  }

  testWidgets('a picked row is filled with its own lane colour', (
    tester,
  ) async {
    await setUpTable(tester);

    Color? rowFill(bool playerOne, String text) {
      final panel = playerOne
          ? find.byType(TrainerWindow).first
          : find.byType(TrainerWindow).last;
      final row = find.ancestor(
        of: find.descendant(of: panel, matching: find.text(text)),
        matching: find.byType(AnimatedContainer),
      );
      if (row.evaluate().isEmpty) return null;
      final decoration = tester.widget<AnimatedContainer>(row.first).decoration;
      return decoration is BoxDecoration ? decoration.color : null;
    }

    expect(rowFill(true, 'KADABRA'), Gba.crimson);
    expect(rowFill(false, 'MACHOKE'), Gba.leaf);
    // An untouched row in the same window stays unfilled.
    expect(rowFill(true, 'SPOOKY'), Colors.transparent);
    expect(tester.takeException(), isNull);
  });

  testWidgets('the table guides you one step at a time', (tester) async {
    await _loadFonts();
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 600));
    expect(tester.takeException(), isNull);
    expect(find.text('LOAD TWO SAVES'), findsOneWidget);

    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('LOAD OTHER SAVE'), findsOneWidget);

    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('PICK ONE FROM EACH'), findsOneWidget);

    await tester.tap(find.text('KADABRA').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 200));
    expect(find.text('PICK FROM PLAYER 2'), findsOneWidget);

    await tester.tap(find.text('MACHOKE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('TRADE'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('unsupported pair shows losses and cannot continue', (
    tester,
  ) async {
    previewSupported = false;
    previewLosses = ['Gen 1/2 cannot store capture-ball identity.'];
    pending = [_saveA.replaceFirst('"generation":3', '"generation":2'), _saveB];
    await _loadFonts();
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 600));

    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.text('KADABRA').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 200));
    await tester.tap(find.text('MACHOKE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 400));

    await tester.tap(find.text('TRADE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump();
    expect(find.text('TRANSFER CHECK'), findsOneWidget);
    expect(find.text('NO CHANGES WILL BE MADE'), findsOneWidget);
    expect(
      find.textContaining('cannot store capture-ball identity'),
      findsOneWidget,
    );
    expect(find.text('CONTINUE'), findsNothing);
    expect(tradeCalls, 0);
    expect(tester.takeException(), isNull);
  });

  testWidgets('trade animation uses each selected capture ball', (
    tester,
  ) async {
    await setUpTable(tester);
    await openRoom(tester);
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await tester.pump();

    final masterBall = find.descendant(
      of: find.byType(TradeRoom),
      matching: find.byWidgetPredicate(
        (widget) =>
            widget is Image &&
            widget.image ==
                const AssetImage('assets/images/balls/master-ball.png'),
      ),
    );
    final premierBall = find.descendant(
      of: find.byType(TradeRoom),
      matching: find.byWidgetPredicate(
        (widget) =>
            widget is Image &&
            widget.image ==
                const AssetImage('assets/images/balls/premier-ball.png'),
      ),
    );
    expect(masterBall, findsOneWidget);
    expect(premierBall, findsOneWidget);
    final topStart = tester.getCenter(masterBall).dy;
    final bottomStart = tester.getCenter(premierBall).dy;

    await tester.pump(const Duration(milliseconds: 1500));
    expect(tester.getCenter(masterBall).dy, greaterThan(topStart));
    expect(tester.getCenter(premierBall).dy, lessThan(bottomStart));
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is ColoredBox &&
            widget.color == Gba.ink.withValues(alpha: 0.46),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('unknown capture ball renders no specific ball sprite', (
    tester,
  ) async {
    pending = [
      _saveA.replaceAll('"ballId":1,', ''),
      _saveB.replaceAll('"ballId":12,', ''),
    ];
    await _loadFonts();
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 500));

    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.text('KADABRA').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 200));
    await tester.tap(find.text('MACHOKE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text('TRADE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump();
    await tester.tap(find.text('CONTINUE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 1200));
    await tester.pump();
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await tester.pump();

    expect(find.byType(CartridgeGlyph), findsNWidgets(2));
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is Image &&
            widget.image is AssetImage &&
            (widget.image as AssetImage).assetName.startsWith(
              'assets/images/balls/',
            ),
      ),
      findsNothing,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('a boxed Pokémon can be picked from the storage screen', (
    tester,
  ) async {
    await _loadFonts();
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 600));

    await tester.tap(find.text('BOXES').first, warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('STORAGE'), findsOneWidget);
    expect(find.text('PIKACHU'), findsOneWidget);

    await tester.tap(find.text('PIKACHU').first, warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('STORAGE'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('a ball from a later generation renders its own sprite', (
    tester,
  ) async {
    pending = [
      _saveA.replaceAll('"ballId":1,', '"ballId":16,'),
      _saveB.replaceAll('"ballId":12,', ''),
    ];
    await _loadFonts();
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 500));

    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('INSERT SAVE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.text('KADABRA').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 200));
    await tester.tap(find.text('MACHOKE').first, warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text('TRADE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump();
    await tester.tap(find.text('CONTINUE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 1200));
    await tester.pump();
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await tester.pump();

    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is Image &&
            widget.image ==
                const AssetImage('assets/images/balls/cherish-ball.png'),
      ),
      findsWidgets,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('successful trade reports completion and clears stale previews', (
    tester,
  ) async {
    await setUpTable(tester);
    await openRoom(tester);
    expect(find.text('YES'), findsOneWidget);

    await tester.tap(find.text('YES'), warnIfMissed: false);
    await runResult(tester);

    expect(tester.takeException(), isNull);
    expect(tradeCalls, 1);
    expect(find.text('TRADE COMPLETE'), findsOneWidget);

    await tester.tap(find.text('OK'), warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 600));
    expect(tester.takeException(), isNull);
    // Previews were cleared, so both bays are empty again.
    expect(find.text('INSERT SAVE'), findsNWidgets(2));
  });

  testWidgets('cancelling the confirm prompt writes nothing', (tester) async {
    await setUpTable(tester);
    await openRoom(tester);
    await tester.tap(find.text('NO'), warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 600));

    expect(tradeCalls, 0);
    expect(tester.takeException(), isNull);
    // Back on the table with both picks intact.
    expect(find.text('TRADE'), findsOneWidget);
  });

  testWidgets('engine refusal surfaces the failure state', (tester) async {
    tradeResult = false;
    await setUpTable(tester);
    await openRoom(tester);
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await runResult(tester);

    expect(tester.takeException(), isNull);
    expect(find.text('TRADE FAILED'), findsOneWidget);
    expect(find.textContaining('DID NOT REPORT'), findsOneWidget);
  });

  testWidgets('engine exception surfaces the failure state', (tester) async {
    tradeResult = 'throw';
    await setUpTable(tester);
    await openRoom(tester);
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await runResult(tester);

    expect(tester.takeException(), isNull);
    expect(find.text('TRADE FAILED'), findsOneWidget);
    expect(find.textContaining('CHECKSUM MISMATCH'), findsOneWidget);
  });

  for (final size in const [
    Size(320, 568), // small phone
    Size(390, 844), // common phone
    Size(820, 1180), // tablet
  ]) {
    testWidgets('lays out without overflow at ${size.width}x${size.height}', (
      tester,
    ) async {
      await setUpTable(tester, size: size);
      expect(tester.takeException(), isNull);
      expect(find.byType(TrainerWindow), findsNWidgets(2));
      expect(find.text('TRADE'), findsOneWidget);
    });
  }

  testWidgets(
    'collapses the reveal when the platform asks for reduced motion',
    (tester) async {
      tester.platformDispatcher.accessibilityFeaturesTestValue =
          const FakeAccessibilityFeatures(disableAnimations: true);
      addTearDown(
        tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
      );

      await setUpTable(tester);
      await openRoom(tester);
      expect(find.text('YES'), findsOneWidget);
      await tester.tap(find.text('YES'), warnIfMissed: false);
      await tester.pump(const Duration(milliseconds: 400));

      expect(tester.takeException(), isNull);
      expect(find.text('TRADE COMPLETE'), findsOneWidget);
    },
  );
}
