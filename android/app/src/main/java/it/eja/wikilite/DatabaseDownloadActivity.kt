package it.eja.wikilite

import android.app.ProgressDialog
import android.content.Intent
import android.content.SharedPreferences
import android.os.AsyncTask
import android.os.Bundle
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.recyclerview.widget.LinearLayoutManager
import kotlinx.coroutines.*
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.io.*
import java.net.URL
import java.util.zip.GZIPInputStream

class DatabaseDownloadActivity : AppCompatActivity() {

    private lateinit var recyclerView: androidx.recyclerview.widget.RecyclerView
    private lateinit var adapter: DatabaseFileAdapter
    private val databaseFiles = mutableListOf<String>()
    private lateinit var preferences: SharedPreferences
    private var progressDialog: ProgressDialog? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_database_download)

        preferences = getSharedPreferences("app_prefs", MODE_PRIVATE)

        setupUI()
        loadDatabaseFiles()
    }

    private fun setupUI() {
        recyclerView = findViewById(R.id.recyclerView)
        recyclerView.layoutManager = LinearLayoutManager(this)

        adapter = DatabaseFileAdapter(databaseFiles) { filePath ->
            showDownloadOptions(filePath)
        }
        recyclerView.adapter = adapter
    }

    private fun loadDatabaseFiles() {
        CoroutineScope(Dispatchers.Main).launch {
            progressDialog = ProgressDialog(this@DatabaseDownloadActivity).apply {
                setMessage("Loading database files...")
                setCancelable(false)
                show()
            }

            val files = withContext(Dispatchers.IO) {
                loadFilesFromHuggingFace()
            }

            progressDialog?.dismiss()
            databaseFiles.clear()
            databaseFiles.addAll(files)
            adapter.notifyDataSetChanged()
        }
    }

    private fun loadFilesFromHuggingFace(): List<String> {
        val files = mutableListOf<String>()
        try {
            val client = OkHttpClient()
            val request = Request.Builder()
                .url("https://huggingface.co/api/datasets/eja/wikilite")
                .build()

            val response = client.newCall(request).execute()
            val jsonResponse = response.body?.string()

            if (jsonResponse != null) {
                val jsonObject = JSONObject(jsonResponse)
                val siblings = jsonObject.getJSONArray("siblings")

                for (i in 0 until siblings.length()) {
                    val item = siblings.getJSONObject(i)
                    val rfilename = item.getString("rfilename")

                    if (rfilename.endsWith(".db.gz")) {
                        files.add(rfilename)
                    }
                }
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
        return files
    }

    private fun getAppStorageLocations(): List<String> {
        val locationPaths = mutableListOf<String>()
        val storageDirs = applicationContext.getExternalFilesDirs(null).filterNotNull()

        if (storageDirs.isNotEmpty()) {
            locationPaths.addAll(storageDirs.map { it.absolutePath })
        }

        if (locationPaths.isEmpty()) {
            locationPaths.add(applicationContext.filesDir.absolutePath)
        }
        return locationPaths
    }

    private fun showDownloadOptions(filePath: String) {
        val storageLocations = getAppStorageLocations()

        if (storageLocations.size <= 1) {
            val downloadPath = storageLocations.first()
            Toast.makeText(this, "Downloading to app storage...", Toast.LENGTH_SHORT).show()
            startDownload(filePath, downloadPath)
        } else {
            val dialog = DownloadDialog(filePath) { selectedFilePath, downloadPath ->
                startDownload(selectedFilePath, downloadPath)
            }
            dialog.show(supportFragmentManager, "download_dialog")
        }
    }

    private fun startDownload(filePath: String, downloadPath: String) {
        DownloadAndExtractTask().execute(filePath to downloadPath)
    }

    private inner class DownloadAndExtractTask : AsyncTask<Pair<String, String>, Long, Boolean>() {
        private lateinit var currentFilePath: String
        private lateinit var finalDbPath: String
        private lateinit var downloadPath: String

        override fun onPreExecute() {
            super.onPreExecute()
            progressDialog = ProgressDialog(this@DatabaseDownloadActivity).apply {
                setMessage("Preparing download...")
                setCancelable(false)
                setProgressStyle(ProgressDialog.STYLE_HORIZONTAL)
                isIndeterminate = false
                show()
            }
        }

        override fun doInBackground(vararg params: Pair<String, String>): Boolean {
            if (params.isEmpty()) return false

            currentFilePath = params[0].first
            downloadPath = params[0].second

            return try {
                val url = URL("https://huggingface.co/datasets/eja/wikilite/resolve/main/$currentFilePath")
                val connection = url.openConnection()
                connection.connect()

                val fileLength = connection.contentLength.toLong()

                val finalFileName = currentFilePath.substringAfterLast("/").replace(".gz", "")
                val outputFile = File(downloadPath, finalFileName)
                finalDbPath = outputFile.absolutePath

                val inputStream = connection.getInputStream()
                var totalExtractedBytes = 0L

                val countingInputStream = object : FilterInputStream(inputStream) {
                    var bytesDownloaded = 0L
                    override fun read(b: ByteArray, off: Int, len: Int): Int {
                        val result = super.read(b, off, len)
                        if (result != -1) {
                            bytesDownloaded += result
                            publishProgress(bytesDownloaded, fileLength, totalExtractedBytes)
                        }
                        return result
                    }
                }

                FileOutputStream(outputFile).use { fos ->
                    GZIPInputStream(countingInputStream).use { gis ->
                        val buffer = ByteArray(8192)
                        var bytesRead: Int
                        while (gis.read(buffer).also { bytesRead = it } != -1) {
                            fos.write(buffer, 0, bytesRead)
                            totalExtractedBytes += bytesRead
                        }
                    }
                }
                true

            } catch (e: Exception) {
                println("Download/Extraction error: ${e.message}")
                e.printStackTrace()
                false
            }
        }

        override fun onProgressUpdate(vararg values: Long?) {
            super.onProgressUpdate(*values)
            val downloaded = values[0] ?: 0L
            val totalDownload = values[1] ?: -1L
            val extracted = values[2] ?: 0L

            val extractedMB = extracted / (1024 * 1024)

            if (totalDownload > 0) {
                progressDialog?.max = totalDownload.toInt()
                progressDialog?.progress = downloaded.toInt()
                progressDialog?.setMessage("Downloading...")
            } else {
                progressDialog?.setMessage("Extracted: ${extractedMB}MB")
            }
        }

        override fun onPostExecute(success: Boolean) {
            progressDialog?.dismiss()

            if (success) {
                preferences.edit().putString("db_path", finalDbPath).apply()
                Toast.makeText(this@DatabaseDownloadActivity, "Download successful!", Toast.LENGTH_SHORT).show()
                startActivity(Intent(this@DatabaseDownloadActivity, MainActivity::class.java))
                finish()
            } else {
                Toast.makeText(this@DatabaseDownloadActivity, "Download failed!", Toast.LENGTH_LONG).show()
            }
        }
    }
}