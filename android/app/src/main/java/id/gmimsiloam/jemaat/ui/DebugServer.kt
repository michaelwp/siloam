package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.data.DEBUG_SERVERS
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.debugServerUrl

/**
 * Debug builds only (nothing in a release build): which server the app talks
 * to, for testing against a local one. Changing it signs out (the session and
 * data were the old server's), after asking when signed in.
 */
@Composable
fun DebugServerSection(signedIn: Boolean, onChange: (String) -> Unit) {
    if (!BuildConfig.DEBUG) return
    val context = LocalContext.current
    val prefs = remember { Prefs(context) }
    var current by remember { mutableStateOf(prefs.serverUrl) }
    var typed by rememberSaveable { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var confirm by rememberSaveable { mutableStateOf<String?>(null) } // the address to switch to, once confirmed

    fun use(url: String) {
        // "" = the built-in server
        val target = url.ifEmpty { BuildConfig.SERVER_URL.trimEnd('/') }
        if (target == current) return
        if (signedIn) confirm = url else apply(url, onChange) { current = it }
    }

    confirm?.let { url ->
        AlertDialog(
            onDismissRequest = { confirm = null },
            title = { Text("Ganti server?") },
            text = { Text("Anda akan keluar dan data di perangkat ini dihapus; masuk lagi di server ${url.ifEmpty { BuildConfig.SERVER_URL }}.") },
            confirmButton = {
                TextButton(onClick = {
                    confirm = null
                    apply(url, onChange) { current = it }
                }) { Text("Ganti") }
            },
            dismissButton = { TextButton(onClick = { confirm = null }) { Text("Batal") } },
        )
    }

    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.tertiaryContainer),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Server (versi debug)", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Text("Sekarang: $current", style = MaterialTheme.typography.bodyMedium)
            DEBUG_SERVERS.forEach { (url, label) ->
                OutlinedButton(onClick = { use(url) }, enabled = url != current, modifier = Modifier.fillMaxWidth()) {
                    Text("$label\n$url", style = MaterialTheme.typography.bodySmall)
                }
            }
            val builtIn = BuildConfig.SERVER_URL.trimEnd('/')
            if (DEBUG_SERVERS.none { it.first == builtIn }) {
                OutlinedButton(onClick = { use("") }, enabled = builtIn != current, modifier = Modifier.fillMaxWidth()) {
                    Text("Bawaan build ini\n$builtIn", style = MaterialTheme.typography.bodySmall)
                }
            }
            val release = BuildConfig.RELEASE_SERVER_URL.trimEnd('/')
            if (release != builtIn && DEBUG_SERVERS.none { it.first == release }) {
                OutlinedButton(onClick = { use(release) }, enabled = release != current, modifier = Modifier.fillMaxWidth()) {
                    Text("Server produksi\n$release", style = MaterialTheme.typography.bodySmall)
                }
            }
            val submit = {
                val url = debugServerUrl(typed)
                if (url == null) {
                    error = "Alamat harus seperti http://192.168.1.5:8080"
                } else {
                    error = null
                    use(url)
                }
            }
            OutlinedTextField(
                value = typed,
                onValueChange = {
                    typed = it
                    error = null
                },
                label = { Text("Alamat lain") },
                placeholder = { Text("http://192.168.1.5:8080") },
                singleLine = true,
                isError = error != null,
                supportingText = error?.let { { Text(it) } },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { submit() }),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedButton(onClick = submit, enabled = typed.isNotBlank()) { Text("Gunakan alamat ini") }
            Text(
                "localhost: jalankan `adb reverse tcp:8080 tcp:8080` (emulator atau HP lewat USB). " +
                    "Hanya ada di versi debug.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun apply(url: String, onChange: (String) -> Unit, shown: (String) -> Unit) {
    onChange(url)
    shown(url.ifEmpty { BuildConfig.SERVER_URL.trimEnd('/') })
}
