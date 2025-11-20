package it.eja.wikilite

import android.annotation.SuppressLint
import android.content.Intent
import android.content.SharedPreferences
import android.os.Bundle
import android.util.Log
import android.view.View
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.ProgressBar
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import java.io.File

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private lateinit var progressBar: ProgressBar
    private lateinit var preferences: SharedPreferences
    private val WIKILITE_LIBRARY_NAME = "libwikilite.so"

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        preferences = getSharedPreferences("app_prefs", MODE_PRIVATE)

        val dbPath = preferences.getString("db_path", "")
        if (dbPath.isNullOrEmpty() || !File(dbPath).exists()) {
            startActivity(Intent(this, DatabaseDownloadActivity::class.java))
            finish()
            return
        }

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
                Log.d("BackButton", "Back pressed - Current URL: ${webView.url}")
                Log.d("BackButton", "Can go back: ${webView.canGoBack()}")

                if (webView.canGoBack()) {
                    Log.d("BackButton", "Calling webView.goBack()")
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

            if (process.isAlive) {
                Log.d("MainActivity", "wikilite process started successfully")
                Thread {
                    process.waitFor()
                }.start()
            } else {
                Log.e("MainActivity", "wikilite process failed to start")
            }

        } catch (e: Exception) {
            Log.e("MainActivity", "An exception occurred while starting the subprocess.", e)
        }
    }
}