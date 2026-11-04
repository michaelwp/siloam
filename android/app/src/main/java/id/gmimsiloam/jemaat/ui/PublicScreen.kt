@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import android.app.Activity
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.ui.platform.LocalContext
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.AppViewModel
import id.gmimsiloam.jemaat.UiState
import id.gmimsiloam.jemaat.data.passwordRules

/**
 * What a visitor (not signed in) sees: the sign-in form and the whole
 * congregation's statistics summary. No jemaat data, no filters, no
 * notifications.
 */
@Composable
fun PublicApp(state: UiState, vm: AppViewModel) {
    // a new or reset account: set its own username and password first, as on the web
    state.mustChange?.let {
        ChangeCredentialsScreen(it, state, vm)
        return
    }
    val context = LocalContext.current
    var confirmClose by rememberSaveable { mutableStateOf(false) }
    if (confirmClose) {
        AlertDialog(
            onDismissRequest = { confirmClose = false },
            title = { Text("Tutup aplikasi?") },
            confirmButton = {
                TextButton(onClick = {
                    confirmClose = false
                    // closes the screens and removes the app from the recent apps
                    (context as? Activity)?.finishAndRemoveTask()
                }) { Text("Tutup") }
            },
            dismissButton = { TextButton(onClick = { confirmClose = false }) { Text("Batal") } },
        )
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("GMIM Siloam", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                        Text("Statistik jemaat", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                },
                actions = {
                    IconButton(onClick = { confirmClose = true }) {
                        Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = "Tutup aplikasi")
                    }
                },
            )
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = state.refreshing,
            onRefresh = vm::refresh,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            Column(
                Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                state.update?.let { UpdateBanner(it) }
                LoginCard(state, vm)
                val stats = state.publicStats
                when {
                    stats != null -> StatsContent(stats.stats, stats.categories)
                    state.refreshing -> Text("Memuat statistik…")
                    else -> Text(
                        "Statistik belum dapat dimuat. Periksa koneksi internet, lalu tarik ke bawah untuk mencoba lagi.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@Composable
private fun LoginCard(state: UiState, vm: AppViewModel) {
    var username by rememberSaveable { mutableStateOf("") }
    var password by rememberSaveable { mutableStateOf("") }
    var server by rememberSaveable { mutableStateOf<String?>(null) } // shown only when asked for
    val submit = { if (username.isNotBlank() && password.isNotEmpty()) vm.login(username, password) }
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Masuk", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Text(
                "Data jemaat, jadwal ibadah dan notifikasi hanya tersedia setelah masuk. Gunakan akun yang sama dengan di web.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedTextField(
                value = username,
                onValueChange = { username = it },
                label = { Text("Username") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Ascii, imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Password") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { submit() }),
                modifier = Modifier.fillMaxWidth(),
            )
            state.loginError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium) }
            Button(
                onClick = submit,
                enabled = !state.loggingIn && username.isNotBlank() && password.isNotEmpty(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(if (state.loggingIn) "Masuk…" else "Masuk") }
            val current = server
            if (current == null) {
                TextButton(onClick = { server = vm.serverUrl() }) { Text("Alamat server") }
            } else {
                OutlinedTextField(
                    value = current,
                    onValueChange = { server = it },
                    label = { Text("Alamat server") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                    modifier = Modifier.fillMaxWidth(),
                )
                TextButton(onClick = {
                    vm.setServerUrl(current)
                    server = null
                }) { Text("Simpan alamat server") }
            }
        }
    }
}

/**
 * Shown after signing in with a temporary password (a new or reset account),
 * like the web's "Perbarui Akun": the user sets their own username and
 * password before going on. Keluar goes back to the sign-in form.
 */
@Composable
private fun ChangeCredentialsScreen(username0: String, state: UiState, vm: AppViewModel) {
    var username by rememberSaveable { mutableStateOf(username0) }
    var current by rememberSaveable { mutableStateOf("") }
    var newPassword by rememberSaveable { mutableStateOf("") }
    var confirm by rememberSaveable { mutableStateOf("") }
    val rules = passwordRules(newPassword, username)
    val valid = rules.all { it.ok } && username.trim().length >= 3 && current.isNotEmpty() && newPassword == confirm
    androidx.activity.compose.BackHandler(onBack = vm::cancelChange)
    Scaffold(
        topBar = { TopAppBar(title = { Text("Perbarui Akun") }) },
    ) { padding ->
        Column(
            Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("Halo $username0, demi keamanan silakan ganti username dan password sementara sebelum melanjutkan.")
            OutlinedTextField(
                value = username,
                onValueChange = { username = it },
                label = { Text("Username") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Ascii, imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth(),
            )
            PasswordField("Password saat ini (sementara)", current) { current = it }
            PasswordField("Password baru", newPassword) { newPassword = it }
            PasswordField("Ulangi password baru", confirm) { confirm = it }
            if (confirm.isNotEmpty() && confirm != newPassword) {
                Text("Konfirmasi password tidak sama.", color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }
            // the rules, ticked live (the byte limit only when broken), as on the web
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                rules.filter { it.id != "max" || !it.ok }.forEach { r ->
                    Text(
                        (if (r.ok) "✓ " else "✗ ") + r.label,
                        style = MaterialTheme.typography.bodySmall,
                        color = if (r.ok) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            state.loginError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium) }
            Button(
                onClick = { vm.changeCredentials(current, username, newPassword) },
                enabled = valid && !state.loggingIn,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(if (state.loggingIn) "Menyimpan…" else "Simpan & Lanjutkan") }
            TextButton(onClick = vm::cancelChange) { Text("Keluar") }
        }
    }
}

@Composable
private fun PasswordField(label: String, value: String, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        singleLine = true,
        visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
        modifier = Modifier.fillMaxWidth(),
    )
}
