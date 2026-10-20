package id.gmimsiloam.jemaat

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.mutableStateOf
import id.gmimsiloam.jemaat.notify.DailyWorker
import id.gmimsiloam.jemaat.notify.Notifier
import id.gmimsiloam.jemaat.ui.SiloamApp
import id.gmimsiloam.jemaat.ui.SiloamTheme

class MainActivity : ComponentActivity() {
    /** The tab a tapped notification asks for. */
    private val requestedTab = mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        Notifier.createChannels(this)
        DailyWorker.schedule(this)
        requestedTab.value = intent?.getStringExtra(Notifier.EXTRA_TAB)
        setContent {
            SiloamTheme {
                SiloamApp(requestedTab = requestedTab.value, onTabHandled = { requestedTab.value = null })
            }
        }
    }

    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        requestedTab.value = intent.getStringExtra(Notifier.EXTRA_TAB)
    }
}
