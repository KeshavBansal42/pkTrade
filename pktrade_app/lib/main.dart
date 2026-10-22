import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'theme.dart';
import 'trade_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.light,
      systemNavigationBarColor: Gba.shellDark,
      systemNavigationBarIconBrightness: Brightness.light,
    ),
  );
  runApp(const PkTradeApp());
}

class PkTradeApp extends StatelessWidget {
  const PkTradeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'pkTrade',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        brightness: Brightness.light,
        fontFamily: 'Silkscreen',
        scaffoldBackgroundColor: Gba.shell,
        splashFactory: NoSplash.splashFactory,
        colorScheme: const ColorScheme.light(
          primary: Gba.action,
          secondary: Gba.leaf,
          surface: Gba.window,
          error: Gba.crimson,
        ),
      ),
      home: const TradeScreen(),
    );
  }
}
