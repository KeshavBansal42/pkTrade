// Visual preview harness, not part of the default suite (it is deliberately not
// named *_test.dart). It drives the real screen with engine-shaped payloads and
// writes PNGs of every state into test/goldens/.
//
//   flutter test --update-goldens test/preview.dart   # refresh the renders
//   open test/goldens/01_empty.png                    # look at them
//
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pktrade_app/trade_screen.dart';

const _channel = MethodChannel('com.pktrade/bridge');

/// Engine-shaped payloads. Player 1 leads with a nicknamed Kadabra, player 2
/// with a Machoke, so the roster exercises the nickname and species paths.
const _saveA =
    '{"generation":3,"trainerName":"UniversalTrainer","tid":0,"sid":0,'
    '"party":['
    '{"index":1,"species":1,"level":5,"nickname":"A","otName":"AAAAA","otId":33403},'
    '{"index":2,"species":64,"level":16,"ballId":1,"nickname":"KADABRA","otName":"AAAAA","otId":33403},'
    '{"index":3,"species":93,"level":22,"nickname":"SPOOKY","otName":"AAAAA","otId":33403}'
    ']}';

const _saveB =
    '{"generation":3,"trainerName":"UniversalTrainer","tid":0,"sid":0,'
    '"party":['
    '{"index":1,"species":7,"level":5,"nickname":"B","otName":"EE","otId":22615},'
    '{"index":2,"species":67,"level":18,"ballId":12,"nickname":"MACHOKE","otName":"EE","otId":22615},'
    '{"index":3,"species":75,"level":24,"nickname":"ROCKY","otName":"EE","otId":22615}'
    ']}';

Future<void> _loadFonts() async {
  const fonts = {'Silkscreen': 'assets/fonts/Silkscreen-Regular.ttf'};
  for (final entry in fonts.entries) {
    final loader = FontLoader(entry.key)..addFont(rootBundle.load(entry.value));
    await loader.load();
  }
  final bold = FontLoader('Silkscreen')
    ..addFont(rootBundle.load('assets/fonts/Silkscreen-Bold.ttf'));
  await bold.load();
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late List<String> pending;

  setUp(() {
    pending = [_saveA, _saveB];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, (call) async {
          switch (call.method) {
            case 'selectSave':
              if (pending.isEmpty) return null;
              return {
                'uri': 'content://save/${pending.length}',
                'json': pending.removeAt(0),
              };
            case 'previewTrade':
              return '{"supported":true,"lossy":false,'
                  '"message":"Verified Gen 3 record swap.","losses":[]}';
            case 'performTrade':
              return true;
          }
          return null;
        });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, null);
  });

  testWidgets('renders the trade flow', (tester) async {
    await _loadFonts();

    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(const MaterialApp(home: TradeScreen()));
    await tester.pump(const Duration(milliseconds: 900));
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/01_empty.png'),
    );

    // Load both saves through the bridge.
    await tester.tap(find.text('INSERT SAVE').first);
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.text('INSERT SAVE').first);
    await tester.pump(const Duration(milliseconds: 900));
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/02_loaded.png'),
    );

    // Pick one Pokemon on each side.
    await tester.tap(find.text('KADABRA').first);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('MACHOKE').first);
    await tester.pump(const Duration(milliseconds: 700));
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/03_selected.png'),
    );

    // Open the trade room and let the prompt type itself out.
    await tester.tap(find.text('TRADE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump();
    await tester.tap(find.text('CONTINUE'), warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 1100));
    await tester.pump();
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/04_confirm.png'),
    );

    // Commit and catch the swap mid-flight.
    await tester.tap(find.text('YES'), warnIfMissed: false);
    await tester.pump(); // let the swap ticker establish its start time
    await tester.pump(const Duration(milliseconds: 800));
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/05_swap.png'),
    );

    // Let it settle into the result.
    await tester.pump(const Duration(milliseconds: 1600));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 700));
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(TradeScreen),
      matchesGoldenFile('goldens/06_result.png'),
    );
  });
}
