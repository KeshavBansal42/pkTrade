import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'dart:convert';

void main() {
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
        brightness: Brightness.dark,
        scaffoldBackgroundColor: const Color(0xFF0F172A),
        primaryColor: const Color(0xFF10B981),
        colorScheme: const ColorScheme.dark(
          primary: Color(0xFF10B981),
          secondary: Color(0xFFE11D48),
        ),
        fontFamily: 'Roboto',
      ),
      home: const TradeScreen(),
    );
  }
}

class TradeScreen extends StatefulWidget {
  const TradeScreen({super.key});

  @override
  State<TradeScreen> createState() => _TradeScreenState();
}

class _TradeScreenState extends State<TradeScreen> {
  static const platform = MethodChannel('com.pktrade/bridge');

  Map<String, dynamic>? saveAData;
  Map<String, dynamic>? saveBData;
  String? uriA;
  String? uriB;

  int? selectedIdxA; // 1-6
  int? selectedIdxB; // 1-6

  bool isTrading = false;

  Future<void> selectSaveFile(bool isSaveA) async {
    try {
      final Map<Object?, Object?>? result = await platform.invokeMethod('selectSave');
      if (result != null) {
        final uri = result['uri'] as String;
        final jsonStr = result['json'] as String;
        final data = jsonDecode(jsonStr);

        setState(() {
          if (isSaveA) {
            saveAData = data;
            uriA = uri;
            selectedIdxA = null;
          } else {
            saveBData = data;
            uriB = uri;
            selectedIdxB = null;
          }
        });
      }
    } on PlatformException catch (e) {
      debugPrint("Failed to select save: '${e.message}'.");
    }
  }

  Future<void> performTrade() async {
    if (uriA == null || uriB == null || selectedIdxA == null || selectedIdxB == null) return;

    setState(() {
      isTrading = true;
    });

    try {
      final success = await platform.invokeMethod('performTrade', {
        'uriA': uriA,
        'uriB': uriB,
        'idxA': selectedIdxA,
        'idxB': selectedIdxB,
      });

      if (success) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('Trade Successful! Files overwritten in place.'),
              backgroundColor: Color(0xFF10B981),
            ),
          );
          // Reload saves to show updated party
          setState(() {
            saveAData = null;
            saveBData = null;
            uriA = null;
            uriB = null;
            selectedIdxA = null;
            selectedIdxB = null;
          });
        }
      }
    } on PlatformException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Trade failed: ${e.message}'),
            backgroundColor: Colors.red,
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          isTrading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('pkTrade Link Cable', style: TextStyle(fontWeight: FontWeight.bold)),
        backgroundColor: Colors.transparent,
        elevation: 0,
        centerTitle: true,
      ),
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: Column(
                children: [
                  Expanded(child: buildSaveColumn(true)),
                  Container(height: 1, color: Colors.white24),
                  Expanded(child: buildSaveColumn(false)),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(24.0),
              child: ElevatedButton(
                onPressed: (selectedIdxA != null && selectedIdxB != null && !isTrading)
                    ? performTrade
                    : null,
                style: ElevatedButton.styleFrom(
                  backgroundColor: const Color(0xFF10B981),
                  disabledBackgroundColor: Colors.white10,
                  minimumSize: const Size(double.infinity, 60),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
                  elevation: selectedIdxA != null && selectedIdxB != null ? 8 : 0,
                ),
                child: isTrading
                    ? const CircularProgressIndicator(color: Colors.white)
                    : const Text(
                        'INITIATE TRADE',
                        style: TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.w900,
                          letterSpacing: 2.0,
                          color: Colors.white,
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget buildSaveColumn(bool isSaveA) {
    final data = isSaveA ? saveAData : saveBData;
    final selectedIdx = isSaveA ? selectedIdxA : selectedIdxB;
    final title = isSaveA ? "Player 1" : "Player 2";

    if (data == null) {
      return Center(
        child: OutlinedButton.icon(
          onPressed: () => selectSaveFile(isSaveA),
          icon: const Icon(Icons.folder_open, color: Colors.white70),
          label: Text('Select $title Save', style: const TextStyle(color: Colors.white70)),
          style: OutlinedButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
            side: const BorderSide(color: Colors.white24, width: 2),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
          ),
        ),
      );
    }

    final party = data['party'] as List<dynamic>;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title.toUpperCase(),
                style: const TextStyle(color: Colors.white54, fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 1.5),
              ),
              const SizedBox(height: 4),
              Text(
                data['trainerName'],
                style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w800),
              ),
              Text(
                "TID: ${data['tid'].toString().padLeft(5, '0')}",
                style: const TextStyle(color: Colors.white54, fontSize: 14),
              ),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: const Color(0xFF10B981).withOpacity(0.2),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: const Color(0xFF10B981), width: 1),
                ),
                child: Text(
                  "Gen ${data['generation'] ?? 3}",
                  style: const TextStyle(color: Color(0xFF10B981), fontSize: 12, fontWeight: FontWeight.bold),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            itemCount: party.length,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            itemBuilder: (context, index) {
              final p = party[index];
              final isSelected = selectedIdx == p['index'];
              
              return GestureDetector(
                onTap: () {
                  setState(() {
                    if (isSaveA) {
                      selectedIdxA = p['index'];
                    } else {
                      selectedIdxB = p['index'];
                    }
                  });
                },
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  margin: const EdgeInsets.only(bottom: 12),
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: isSelected ? const Color(0xFF10B981).withOpacity(0.15) : Colors.white.withOpacity(0.05),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(
                      color: isSelected ? const Color(0xFF10B981) : Colors.transparent,
                      width: 2,
                    ),
                    boxShadow: isSelected
                        ? [
                            BoxShadow(
                              color: const Color(0xFF10B981).withOpacity(0.2),
                              blurRadius: 12,
                              spreadRadius: 2,
                            )
                          ]
                        : [],
                  ),
                  child: Row(
                    children: [
                      Container(
                        width: 48,
                        height: 48,
                        decoration: BoxDecoration(
                          color: Colors.white10,
                          shape: BoxShape.circle,
                        ),
                        child: Center(
                          child: Text(
                            "Lv${p['level']}",
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12, color: Colors.white70),
                          ),
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              p['nickname'],
                              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                            const SizedBox(height: 4),
                            Text(
                              "Species ID: ${p['species']}",
                              style: const TextStyle(fontSize: 12, color: Colors.white54),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }
}
