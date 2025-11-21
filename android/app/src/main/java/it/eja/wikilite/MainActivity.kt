package it.eja.wikilite

import android.Manifest
import android.annotation.SuppressLint
import android.content.Intent
import android.content.SharedPreferences
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.util.Log
import android.view.View
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.ProgressBar
import android.widget.Toast
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import java.io.File

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private lateinit var progressBar: ProgressBar
    private lateinit var preferences: SharedPreferences
    private val WIKILITE_LIBRARY_NAME = "libwikilite.so"
    private val DB_FILENAME = "wikilite.db"
    private val PERMISSION_REQUEST_CODE = 100

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        if (hasExternalSdCard()) {
            if (!checkStoragePermissions()) {
                return
            }
        }

        initializeAppLogic()
    }

    private fun hasExternalSdCard(): Boolean {
        val dirs = getExternalFilesDirs(null)
        return dirs.size > 1
    }

    private fun initializeAppLogic() {
        preferences = getSharedPreferences("app_prefs", MODE_PRIVATE)

        val discoveredDb = discoverDatabase()

        if (discoveredDb != null) {
            preferences.edit().putString("db_path", discoveredDb.absolutePath).apply()
            initApp(discoveredDb.absolutePath)
        } else {
            val savedDbPath = preferences.getString("db_path", "")
            if (!savedDbPath.isNullOrEmpty() && File(savedDbPath).exists() && savedDbPath.endsWith(DB_FILENAME)) {
                initApp(savedDbPath)
            } else {
                startActivity(Intent(this, DatabaseDownloadActivity::class.java))
                finish()
            }
        }
    }

    private fun checkStoragePermissions(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                try {
                    val intent = Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION)
                    intent.addCategory("android.intent.category.DEFAULT")
                    intent.data = Uri.parse(String.format("package:%s", applicationContext.packageName))
                    startActivityForResult(intent, PERMISSION_REQUEST_CODE)
                } catch (e: Exception) {
                    val intent = Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION)
                    startActivityForResult(intent, PERMISSION_REQUEST_CODE)
                }
                return false
            }
        } else {
            val permissions = arrayOf(
                Manifest.permission.READ_EXTERNAL_STORAGE,
                Manifest.permission.WRITE_EXTERNAL_STORAGE
            )
            val missingPermissions = permissions.filter {
                ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
            }
            if (missingPermissions.isNotEmpty()) {
                ActivityCompat.requestPermissions(this, missingPermissions.toTypedArray(), PERMISSION_REQUEST_CODE)
                return false
            }
        }
        return true
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == PERMISSION_REQUEST_CODE) {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                if (Environment.isExternalStorageManager()) {
                    initializeAppLogic()
                } else {
                    Toast.makeText(this, "Storage permission is required to check the SD card.", Toast.LENGTH_LONG).show()
                    initializeAppLogic()
                }
            }
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == PERMISSION_REQUEST_CODE) {
            if (grantResults.isNotEmpty() && grantResults.all { it == PackageManager.PERMISSION_GRANTED }) {
                initializeAppLogic()
            } else {
                Toast.makeText(this, "Storage permission is required to check the SD card.", Toast.LENGTH_LONG).show()
                initializeAppLogic()
            }
        }
    }

    private fun discoverDatabase(): File? {
        val externalDirs = getExternalFilesDirs(null)
        for (dir in externalDirs) {
            if (dir != null && dir != filesDir) {
                val pathStr = dir.absolutePath
                val rootPath = if (pathStr.contains("/Android")) {
                    pathStr.substringBefore("/Android")
                } else {
                    pathStr
                }

                val externalRootFile = File(rootPath, DB_FILENAME)
                if (externalRootFile.exists()) {
                    return externalRootFile
                }
            }
        }

        val internalFile = File(filesDir, DB_FILENAME)
        if (internalFile.exists()) return internalFile

        return null
    }

    @SuppressLint("SetJavaScriptEnabled")
    private fun initApp(dbPath: String) {
        webView = findViewById(R.id.webView)
        progressBar = findViewById(R.id.progressBar)
        setupWebView()
        handleBackPress()

        try {
            Thread {
                startWikiLiteProcess(dbPath)
            }.start()

            webView.postDelayed({
                webView.loadUrl("http://127.0.0.1:35248/")
            }, 5000)

        } catch (e: Exception) {
            Log.e("MainActivity", "Failed to setup and run wikilite", e)
            val errorMessage = "<html><body><h1>Error</h1><p>${e.message}</p></body></html>"
            webView.loadData(errorMessage, "text/html", "UTF-8")
        }
    }

    @SuppressLint("SetJavaScriptEnabled")
    private fun setupWebView() {
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.settings.allowFileAccess = true
        webView.settings.allowContentAccess = true

        webView.webViewClient = object : WebViewClient() {
            override fun onPageStarted(view: WebView?, url: String?, favicon: android.graphics.Bitmap?) {
                super.onPageStarted(view, url, favicon)
                progressBar.visibility = View.VISIBLE
                webView.visibility = View.INVISIBLE
            }

            override fun onPageFinished(view: WebView, url: String) {
                super.onPageFinished(view, url)
                progressBar.visibility = View.GONE
                webView.visibility = View.VISIBLE
                Log.d("WebView", "Page finished: $url")
                Log.d("WebView", "Can go back: ${view.canGoBack()}")
            }
        }
    }

    private fun handleBackPress() {
        val callback = object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (webView.canGoBack()) {
                    webView.goBack()
                } else {
                    isEnabled = false
                    onBackPressedDispatcher.onBackPressed()
                }
            }
        }
        onBackPressedDispatcher.addCallback(this, callback)
    }

    private fun startWikiLiteProcess(dbPath: String) {
        try {
            val cwd = cacheDir
            cwd.mkdirs()

            val executablePath = File(applicationInfo.nativeLibraryDir, WIKILITE_LIBRARY_NAME).absolutePath
            val libraryPath = applicationInfo.nativeLibraryDir

            val command = arrayOf(
                executablePath,
                "--db", dbPath,
                "--web",
                "--web-port", "35248",
                "--web-host", "0.0.0.0"
            )

            Log.d("MainActivity", "Executing command: ${command.joinToString(" ")}")

            val processBuilder = ProcessBuilder(*command)
                .directory(cwd)
                .redirectErrorStream(true)

            val env = processBuilder.environment()
            env["LD_LIBRARY_PATH"] = libraryPath
            env["HOME"] = cwd.absolutePath
            env["TMPDIR"] = cwd.absolutePath
            env["PATH"] = "$libraryPath:${env["PATH"] ?: ""}"

            val process = processBuilder.start()

            Thread {
                val reader = process.inputStream.bufferedReader()
                var line: String?
                while (reader.readLine().also { line = it } != null) {
                    Log.d("wikilite", line ?: "")
                }
            }.start()

            Thread.sleep(2000)

            var isAlive = false
            try {
                process.exitValue()
                isAlive = false
            } catch (e: IllegalThreadStateException) {
                isAlive = true
            }

            if (isAlive) {
                Log.d("MainActivity", "wikilite process started successfully")
                Thread {
                    try {
                        process.waitFor()
                    } catch (e: InterruptedException) {
                        e.printStackTrace()
                    }
                }.start()
            } else {
                Log.e("MainActivity", "wikilite process failed to start")
            }

        } catch (e: Exception) {
            Log.e("MainActivity", "An exception occurred while starting the subprocess.", e)
        }
    }
}