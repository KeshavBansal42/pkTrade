package com.example.pktrade_app

import android.app.Activity
import android.content.Intent
import android.net.Uri
import java.io.File
import java.io.FileOutputStream
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.UUID
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
                "previewTrade" -> {
                    try {
                        val uriA = Uri.parse(call.argument<String>("uriA")!!)
                        val uriB = Uri.parse(call.argument<String>("uriB")!!)
                        val idxA = call.argument<Int>("idxA")!!
                        val idxB = call.argument<Int>("idxB")!!
                        val bytesA = contentResolver.openInputStream(uriA)?.readBytes()
                        val bytesB = contentResolver.openInputStream(uriB)?.readBytes()
                        if (bytesA == null || bytesB == null) {
                            error("Could not read both save files")
                        }
                        result.success(Mobile.previewTradeJSON(bytesA, bytesB, idxA.toLong(), idxB.toLong()))
                    } catch (error: Exception) {
                        result.error("PREVIEW_ERROR", error.message, null)
                    }
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
                            require(uriA != uriB) { "Choose two different save files" }

                            val tradeResult = Mobile.performTrade(bytesA, bytesB, idxA.toLong(), idxB.toLong())
                            val backupA = createBackup(bytesA)
                            val backupB = try {
                                createBackup(bytesB)
                            } catch (error: Exception) {
                                backupA.delete()
                                throw error
                            }

                            try {
                                writeDocument(uriA, tradeResult.saveA)
                                writeDocument(uriB, tradeResult.saveB)
                            } catch (error: Exception) {
                                val restoreErrors = listOf(
                                    runCatching { writeDocument(uriA, backupA.readBytes()) }.exceptionOrNull(),
                                    runCatching { writeDocument(uriB, backupB.readBytes()) }.exceptionOrNull()
                                ).filterNotNull()
                                if (restoreErrors.isNotEmpty()) {
                                    throw IllegalStateException(
                                        "Trade write failed and rollback was incomplete. Backups: ${backupA.absolutePath}, ${backupB.absolutePath}",
                                        error
                                    )
                                }
                                throw error
                            }

                            result.success(true)
                        } else {
                            result.error("IO_ERROR", "Could not read save files", null)
                        }
                    } catch (e: Exception) {
                        result.error("TRADE_ERROR", e.message, null)
                    }
                }
                "listSlots" -> {
                    try {
                        val uri = Uri.parse(call.argument<String>("uri")!!)
                        val includeEmpty = call.argument<Boolean>("includeEmpty") ?: false
                        val bytes = contentResolver.openInputStream(uri)?.readBytes()
                        if (bytes == null) {
                            error("Could not read the save file")
                        }
                        result.success(Mobile.listSlotsJSON(bytes, includeEmpty))
                    } catch (error: Exception) {
                        result.error("SLOTS_ERROR", error.message, null)
                    }
                }
                "previewSlotTrade" -> {
                    try {
                        val uriA = Uri.parse(call.argument<String>("uriA")!!)
                        val uriB = Uri.parse(call.argument<String>("uriB")!!)
                        val slotA = call.argument<String>("slotA")!!
                        val slotB = call.argument<String>("slotB")!!
                        val bytesA = contentResolver.openInputStream(uriA)?.readBytes()
                        val bytesB = contentResolver.openInputStream(uriB)?.readBytes()
                        if (bytesA == null || bytesB == null) {
                            error("Could not read both save files")
                        }
                        result.success(Mobile.previewSlotTradeJSON(bytesA, bytesB, slotA, slotB))
                    } catch (error: Exception) {
                        result.error("PREVIEW_ERROR", error.message, null)
                    }
                }
                "performSlotTrade" -> {
                    try {
                        val uriAStr = call.argument<String>("uriA")!!
                        val uriBStr = call.argument<String>("uriB")!!
                        val slotA = call.argument<String>("slotA")!!
                        val slotB = call.argument<String>("slotB")!!

                        val uriA = Uri.parse(uriAStr)
                        val uriB = Uri.parse(uriBStr)

                        val bytesA = contentResolver.openInputStream(uriA)?.readBytes()
                        val bytesB = contentResolver.openInputStream(uriB)?.readBytes()

                        if (bytesA != null && bytesB != null) {
                            require(uriA != uriB) { "Choose two different save files" }

                            val tradeResult = Mobile.performSlotTrade(bytesA, bytesB, slotA, slotB)
                            val backupA = createBackup(bytesA)
                            val backupB = try {
                                createBackup(bytesB)
                            } catch (error: Exception) {
                                backupA.delete()
                                throw error
                            }

                            try {
                                writeDocument(uriA, tradeResult.saveA)
                                writeDocument(uriB, tradeResult.saveB)
                            } catch (error: Exception) {
                                val restoreErrors = listOf(
                                    runCatching { writeDocument(uriA, backupA.readBytes()) }.exceptionOrNull(),
                                    runCatching { writeDocument(uriB, backupB.readBytes()) }.exceptionOrNull()
                                ).filterNotNull()
                                if (restoreErrors.isNotEmpty()) {
                                    throw IllegalStateException(
                                        "Trade write failed and rollback was incomplete. Backups: ${backupA.absolutePath}, ${backupB.absolutePath}",
                                        error
                                    )
                                }
                                throw error
                            }

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

    private fun createBackup(bytes: ByteArray): File {
        val directory = File(filesDir, "save-backups")
        check(directory.exists() || directory.mkdirs()) { "Could not create save backup directory" }
        val stamp = SimpleDateFormat("yyyyMMdd-HHmmss-SSS", Locale.US).format(Date())
        val backup = File(directory, "save-$stamp-${UUID.randomUUID()}.sav.bak")
        FileOutputStream(backup).use { it.write(bytes) }
        return backup
    }

    private fun writeDocument(uri: Uri, bytes: ByteArray) {
        val stream = contentResolver.openOutputStream(uri, "wt")
            ?: error("Could not open save file for writing")
        stream.use { it.write(bytes) }
        // Read the document back so a provider that silently dropped or
        // truncated the write cannot leave a damaged save behind.
        val written = contentResolver.openInputStream(uri)?.use { it.readBytes() }
            ?: error("Could not read the save file back after writing")
        if (!written.contentEquals(bytes)) {
            error("The save file did not keep the new data")
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
