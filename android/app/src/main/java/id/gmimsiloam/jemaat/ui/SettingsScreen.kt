@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

package id.gmimsiloam.jemaat.ui

import android.app.TimePickerDialog
import android.content.Intent
import android.os.Build
import android.provider.Settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.AppViewModel
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.UiState
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.notify.DailyWorker
import id.gmimsiloam.jemaat.notify.Notifier
import id.gmimsiloam.jemaat.notify.showDaily
import java.text.DateFormat
import java.time.LocalDate
import java.util.Date

private val REMIND_CHOICES = listOf(0 to "Tidak", 15 to "15 menit", 30 to "30 menit", 60 to "1 jam", 120 to "2 jam")

@Composable
fun SettingsScreen(state: UiState, vm: AppViewModel, onBack: () -> Unit) {
    val context = LocalContext.current
    val prefs = remember { Prefs(context) }
    var services by remember { mutableStateOf(prefs.notifyServices) }
    var birthdays by remember { mutableStateOf(prefs.notifyBirthdays) }
    var remind by remember { mutableIntStateOf(prefs.remindBeforeMinutes) }
    var dailyAt by remember { mutableIntStateOf(prefs.dailyAtMinutes) }
    var server by remember { mutableStateOf(prefs.serverUrl) }
    val allowed = Notifier.canPost(context)

    androidx.activity.compose.BackHandler(onBack = onBack)
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pengaturan") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Kembali") }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            DataFilterFields(state, vm::setFilter)
            HorizontalDivider()
            Text("Notifikasi", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            if (!allowed) {
                Text(
                    "Notifikasi untuk aplikasi ini dimatikan di pengaturan Android.",
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodyMedium,
                )
                OutlinedButton(onClick = {
                    val i = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, context.packageName)
                    } else {
                        Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, android.net.Uri.parse("package:${context.packageName}"))
                    }
                    context.startActivity(i)
                }) { Text("Buka pengaturan notifikasi") }
            }
            Toggle("Jadwal ibadah", "Ibadah hari ini dan pengingat sebelum ibadah", services) {
                services = it
                prefs.notifyServices = it
                DailyWorker.rescheduleReminders(context)
            }
            Toggle("Ulang tahun", "Anggota jemaat yang berulang tahun hari ini", birthdays) {
                birthdays = it
                prefs.notifyBirthdays = it
                DailyWorker.runNow(context)
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Ringkasan harian", style = MaterialTheme.typography.bodyLarge)
                    Text("Ditampilkan sekali sehari setelah jam ini", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                OutlinedButton(onClick = {
                    TimePickerDialog(context, { _, h, m ->
                        dailyAt = h * 60 + m
                        prefs.dailyAtMinutes = dailyAt
                        DailyWorker.runNow(context)
                    }, dailyAt / 60, dailyAt % 60, true).show()
                }) { Text("%02d.%02d".format(dailyAt / 60, dailyAt % 60)) }
            }
            Text("Pengingat sebelum ibadah", style = MaterialTheme.typography.bodyLarge)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                REMIND_CHOICES.forEach { (mins, label) ->
                    FilterChip(selected = remind == mins, enabled = services, onClick = {
                        remind = mins
                        prefs.remindBeforeMinutes = mins
                        // today's queued reminders were made with the old time
                        DailyWorker.rescheduleReminders(context)
                    }, label = { Text(label) })
                }
            }
            Text(
                "Aplikasi memeriksa jadwal di latar belakang sekitar setiap jam, juga setelah ponsel dinyalakan ulang.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedButton(onClick = {
                // counts as today's summary, so it does not come again later today
                state.data?.let { showDaily(context, prefs, it, LocalDate.now(), force = true) }
            }, enabled = allowed && state.data != null) { Text("Tampilkan notifikasi hari ini sekarang") }

            HorizontalDivider()
            Text("Server", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            // release builds only talk HTTPS (network_security_config); debug builds
            // may use http:// for a computer on the network or the emulator
            val serverError = serverUrlError(server.trim())
            OutlinedTextField(
                value = server,
                onValueChange = { server = it },
                label = { Text("Alamat server") },
                singleLine = true,
                isError = serverError != null,
                supportingText = serverError?.let { { Text(it) } },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { vm.setServerUrl(server.trim()) }, enabled = serverError == null) { Text("Simpan & muat ulang") }
                OutlinedButton(onClick = { server = Prefs.DEFAULT_SERVER; vm.setServerUrl(Prefs.DEFAULT_SERVER) }) { Text("Bawaan") }
            }
            state.data?.let {
                Text(
                    "Data terakhir diperbarui: " + DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT).format(Date(it.fetchedAt)),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            state.error?.let { Text("Galat terakhir: $it", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error) }

            HorizontalDivider()
            Text("Tentang aplikasi", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Versi", style = MaterialTheme.typography.bodyLarge, modifier = Modifier.weight(1f))
                Text(BuildConfig.VERSION_NAME, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.SemiBold)
            }
            state.update?.let { UpdateBanner(it) }
            Text(
                "Jemaat Siloam · GMIM Siloam. Aplikasi ini hanya menampilkan data publik (tanpa tahun lahir, alamat, atau hubungan keluarga).",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

/**
 * The "filter data bawaan" (the same choice as the top bar): a kolom or all,
 * a BIPRA category or all, and a jenis ibadah or all. The screens and the notifications follow it.
 */
@Composable
private fun DataFilterFields(state: UiState, onPick: (DataFilter) -> Unit) {
    val f = state.filter
    val data = state.data
    val categories = data?.categories.orEmpty()
    val koloms = data?.koloms.orEmpty().map { it.number }
    val kolom = data?.let { f.kolomOf(it) } ?: f.kolom
    val category = data?.let { f.category(it) }

    Text("Filter data bawaan", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
    Text("Kolom", style = MaterialTheme.typography.labelLarge)
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        FilterChip(selected = kolom == null, onClick = { onPick(f.withKolom(null)) }, label = { Text("Semua Kolom") })
        koloms.forEach { k ->
            FilterChip(selected = kolom == k, onClick = { onPick(f.withKolom(k)) }, label = { Text("Kolom $k") })
        }
    }
    if (categories.isNotEmpty()) {
        Text("Kategori BIPRA", style = MaterialTheme.typography.labelLarge)
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(selected = category == null, onClick = { onPick(f.withBipra(null)) }, label = { Text("Semua BIPRA") })
            categories.forEach { c ->
                FilterChip(selected = category == c, onClick = { onPick(f.withBipra(c)) }, label = { Text(c.label) })
            }
        }
    }
    val types = data?.let { f.typeChoices(it) }.orEmpty()
    if (types.isNotEmpty()) {
        val type = data?.let { f.type(it) }
        Text("Jenis Ibadah", style = MaterialTheme.typography.labelLarge)
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(selected = type == null, onClick = { onPick(f.withType(null)) }, label = { Text("Semua Jenis") })
            types.forEach { t ->
                FilterChip(selected = type == t, onClick = { onPick(f.withType(t)) }, label = { Text(t.name) })
            }
        }
        type?.let { Hint("Jadwal dan notifikasi ibadah: Ibadah ${it.name} saja.") }
    }
    Hint(
        when {
            kolom == null && category == null -> "Semua data jemaat, ibadah dan notifikasi (dikelompokkan per kolom)."
            category == null -> "Data Kolom $kolom, ibadahnya dan ibadah untuk seluruh jemaat, dan notifikasinya."
            kolom == null -> "Anggota ${category.label} dan keluarganya, ibadah jenis kategori ini (di semua kolom), dan notifikasinya."
            else -> "Anggota ${category.label} di Kolom $kolom dan keluarganya, ibadah jenis kategori ini di kolom ini (dan untuk seluruh jemaat), dan notifikasinya."
        },
    )
    Hint("Dipakai saat aplikasi dibuka dan untuk notifikasi; dapat juga diubah di bagian atas layar.")
}

@Composable
private fun Hint(text: String) {
    Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun Toggle(title: String, subtitle: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

/** Why [url] cannot be used as the server address, or null when it can. */
internal fun serverUrlError(url: String, debug: Boolean = BuildConfig.DEBUG): String? {
    val ok = url.startsWith("https://") || (debug && url.startsWith("http://"))
    return when {
        url.isEmpty() -> "Isi alamat server"
        !ok && url.startsWith("http://") -> "Gunakan alamat https:// (koneksi tanpa enkripsi tidak diizinkan)"
        !ok -> "Alamat harus diawali https://"
        url.removePrefix("https://").removePrefix("http://").isBlank() -> "Alamat server belum lengkap"
        else -> null
    }
}
