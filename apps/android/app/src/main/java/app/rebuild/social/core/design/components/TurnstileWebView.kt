package app.rebuild.social.core.design.components

import android.annotation.SuppressLint
import android.os.Handler
import android.os.Looper
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

/**
 * Embeds the Cloudflare Turnstile widget in a WebView (Turnstile has no native
 * Android SDK). Emits the token via [onSuccess] or a message via [onError].
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TurnstileWebView(
    siteKey: String,
    onSuccess: (String) -> Unit,
    onError: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val currentOnSuccess by rememberUpdatedState(onSuccess)
    val currentOnError by rememberUpdatedState(onError)
    val bridge = remember {
        TurnstileJsBridge(
            onSuccessCallback = { currentOnSuccess(it) },
            onErrorCallback = { currentOnError(it) }
        )
    }

    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.setSupportMultipleWindows(false)
                addJavascriptInterface(bridge, "LanternTurnstile")
                webViewClient = WebViewClient()
                loadDataWithBaseURL(
                    "https://localhost/",
                    turnstileHtml(siteKey),
                    "text/html",
                    "utf-8",
                    null
                )
            }
        },
        onRelease = { it.destroy() }
    )
}

private class TurnstileJsBridge(
    private val onSuccessCallback: (String) -> Unit,
    private val onErrorCallback: (String) -> Unit
) {
    private val mainHandler = Handler(Looper.getMainLooper())

    @JavascriptInterface
    fun onSuccess(token: String) {
        mainHandler.post { onSuccessCallback(token) }
    }

    @JavascriptInterface
    fun onError(message: String) {
        mainHandler.post { onErrorCallback(message) }
    }
}

private fun turnstileHtml(siteKey: String): String = """
    <!DOCTYPE html>
    <html>
    <head>
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <script src="https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"></script>
    </head>
    <body style="margin:0;display:flex;justify-content:center;padding:4px;">
      <div id="cf-turnstile"></div>
      <script>
        function renderTurnstile() {
          if (!window.turnstile) { setTimeout(renderTurnstile, 200); return; }
          window.turnstile.render('cf-turnstile', {
            sitekey: '$siteKey',
            callback: function (token) { LanternTurnstile.onSuccess(token); },
            'error-callback': function () { LanternTurnstile.onError('challenge_error'); },
            'expired-callback': function () { LanternTurnstile.onError('challenge_expired'); }
          });
        }
        renderTurnstile();
      </script>
    </body>
    </html>
""".trimIndent()
