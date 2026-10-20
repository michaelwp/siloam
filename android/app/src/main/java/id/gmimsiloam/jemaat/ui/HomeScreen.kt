package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.anniversariesToday
import id.gmimsiloam.jemaat.data.birthdaysToday
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.inKolom
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.shortDate
import id.gmimsiloam.jemaat.data.upcomingBirthdays
import id.gmimsiloam.jemaat.data.upcomingServices
import java.text.DateFormat
import java.time.LocalDate
import java.util.Date

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun HomeScreen(data: Snapshot, kolom: Int?, error: String?) {
    val today = remember { LocalDate.now() }
    val services = data.services.inKolom(kolom) { it.kolom }
    val members = data.members.inKolom(kolom) { it.kolom }
    val families = data.families.inKolom(kolom) { it.kolom }
    val up = upcomingServices(services, today)
    val birthdays = birthdaysToday(members, today)
    val nextBirthdays = upcomingBirthdays(members, today)
    val anniversaries = anniversariesToday(families, today)
    var selected by remember { mutableStateOf<Service?>(null) }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        if (error != null) {
            Text(
                "Tidak dapat terhubung ke server; menampilkan data tersimpan dari " +
                    DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT).format(Date(data.fetchedAt)) + ".",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        SectionCard("⛪", "Ibadah Hari Ini", longDate(today)) {
            if (up.today.isEmpty()) Text("Tidak ada ibadah hari ini.", style = MaterialTheme.typography.bodyLarge)
            up.today.forEach { ServiceRow(it, withDate = false, highlight = true) { selected = it } }
            if (up.week.isNotEmpty()) {
                Text("Minggu Ini", style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                up.week.forEach { ServiceRow(it, withDate = true) { selected = it } }
            } else if (up.next != null) {
                Text(
                    "Berikutnya: ${up.next.title} (${shortDate(up.next.date)}, ${clock(up.next.time)})",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }

        SectionCard("🎂", "Ulang Tahun Hari Ini", longDate(today), accent = MaterialTheme.colorScheme.secondary) {
            if (birthdays.isEmpty()) {
                Text("Tidak ada anggota jemaat yang berulang tahun hari ini.", style = MaterialTheme.typography.bodyLarge)
            }
            birthdays.forEach { m ->
                Surface(shape = RoundedCornerShape(10.dp), color = MaterialTheme.colorScheme.secondaryContainer, modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp)) {
                        Text(m.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                        Text("Kel. ${m.familyName}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            if (nextBirthdays.isNotEmpty()) {
                Text("Berikutnya:", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    nextBirthdays.forEach { (m, days) ->
                        Pill(
                            "${m.name} · " + if (days == 1L) "besok" else "${m.birthDay} ${MONTHS[m.birthMonth!! - 1].take(3)}",
                            MaterialTheme.colorScheme.primaryContainer,
                        )
                    }
                }
            }
        }

        if (anniversaries.isNotEmpty()) {
            SectionCard("💍", "HUT Pernikahan Hari Ini", accent = MaterialTheme.colorScheme.tertiary) {
                anniversaries.forEach { Text("Kel. ${it.name}", style = MaterialTheme.typography.titleMedium) }
            }
        }

        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            Stat("Keluarga", families.size, Modifier.weight(1f))
            Stat("Jiwa", members.size, Modifier.weight(1f))
            Stat("Ibadah", services.size, Modifier.weight(1f))
        }
    }

    selected?.let { ServiceSheet(it) { selected = null } }
}

@Composable
private fun Stat(label: String, value: Int, modifier: Modifier) {
    Surface(shape = RoundedCornerShape(12.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 1.dp, modifier = modifier) {
        Column(Modifier.padding(14.dp)) {
            Text("$value", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
