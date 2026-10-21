package id.gmimsiloam.jemaat

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.mutableStateOf
import id.gmimsiloam.jemaat.notify.DailyWorker
import id.gmimsiloam.jemaat.notify.Notifier
import id.gmimsiloam.jemaat.ui.SiloamApp
import id.gmimsiloam.jemaat.ui.SiloamTheme
import id.gmimsiloam.jemaat.ui.SplashOverlay

class MainActivity : ComponentActivity() {
    /** The tab a tapped notification asks for. */
    private val requestedTab = mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        Notifier.createChannels(this)
        DailyWorker.schedule(this)
        // only on a fresh start: after a rotation the user's own tab stays
        if (savedInstanceState == null) requestedTab.value = intent?.getStringExtra(Notifier.EXTRA_TAB)
        // opened from a notification: straight to its tab, no splash
        val fromNotification = requestedTab.value != null
        setContent {
            SiloamTheme {
                Box {
                    SiloamApp(requestedTab = requestedTab.value, onTabHandled = { requestedTab.value = null })
                    SplashOverlay(skip = fromNotification)
                }
            }
        }
    }

    /**
     * Coming to the foreground also runs the check right away: e.g.
     * notifications were just allowed in Android's settings and today's
     * summary is still due. (Cheap: data is only fetched when it is stale.)
     */
    override fun onStart() {
        super.onStart()
        DailyWorker.runNow(this)
    }

    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        requestedTab.value = intent.getStringExtra(Notifier.EXTRA_TAB)
    }
}
