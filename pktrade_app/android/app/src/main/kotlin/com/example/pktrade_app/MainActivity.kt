package com.example.pktrade_app

import android.app.Activity
import android.content.Intent
import android.net.Uri
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import mobile.Mobile

class MainActivity: FlutterActivity() {
    private val CHANNEL = "com.pktrade/bridge"
    private var pendingResult: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler { call, result ->
            when (call.method) {
                "selectSave" -> {
                    pendingResult = result
                    val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                        addCategory(Intent.CATEGORY_OPENABLE)
                        type = "*/*"
                    }
                    startActivityForResult(intent, 1001)
                }
                "performTrade" -> {
                    try {
                        val uriAStr = call.argument<String>("uriA")!!
                        val uriBStr = call.argument<String>("uriB")!!
                        val idxA = call.argument<Int>("idxA")!!
                        val idxB = call.argument<Int>("idxB")!!

                        val uriA = Uri.parse(uriAStr)
                        val uriB = Uri.parse(uriBStr)

                        val bytesA = contentResolver.openInputStream(uriA)?.readBytes()
                        val bytesB = contentResolver.openInputStream(uriB)?.readBytes()

                        if (bytesA != null && bytesB != null) {
                            // Call Go gomobile Library
                            val tradeResult = Mobile.performTrade(bytesA, bytesB, idxA.toLong(), idxB.toLong())
                            
                            // Overwrite files in-place using Android SAF
                            contentResolver.openOutputStream(uriA, "wt")?.use { it.write(tradeResult.saveA) }
                            contentResolver.openOutputStream(uriB, "wt")?.use { it.write(tradeResult.saveB) }

                            result.success(true)
                        } else {
                            result.error("IO_ERROR", "Could not read save files", null)
                        }
                    } catch (e: Exception) {
                        result.error("TRADE_ERROR", e.message, null)
                    }
                }
                else -> result.notImplemented()
            }
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == 1001 && resultCode == Activity.RESULT_OK) {
            data?.data?.let { uri ->
                try {
                    // Take persistable permission so we can write to it later
                    val takeFlags: Int = Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_WRITE_URI_PERMISSION
                    contentResolver.takePersistableUriPermission(uri, takeFlags)

                    // Read bytes and pass to Go for JSON parsing
                    val bytes = contentResolver.openInputStream(uri)?.readBytes()
                    if (bytes != null) {
                        val json = Mobile.parsePartyJSON(bytes)
                        val response = mapOf("uri" to uri.toString(), "json" to json)
                        pendingResult?.success(response)
                    } else {
                        pendingResult?.error("READ_ERROR", "Could not read file", null)
                    }
                } catch (e: Exception) {
                    pendingResult?.error("PARSE_ERROR", e.message, null)
                }
            } ?: run {
                pendingResult?.success(null)
            }
            pendingResult = null
        } else if (requestCode == 1001) {
            pendingResult?.success(null)
            pendingResult = null
        }
    }
}
