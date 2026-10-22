package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.AppRelease
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.MemberFilter
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.anniversariesToday
import id.gmimsiloam.jemaat.data.birthdaysToday
import id.gmimsiloam.jemaat.data.byKolom
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.kolomHeading
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.shortDate
import id.gmimsiloam.jemaat.data.upcomingBirthdays
import id.gmimsiloam.jemaat.data.upcomingServices
import java.text.DateFormat
import java.time.LocalDate
import java.util.Date

@Composable
fun HomeScreen(data: Snapshot, filter: DataFilter, error: String?, update: AppRelease?, onCategory: (String) -> Unit, onMembers: (MemberFilter) -> Unit) {
    val today = rememberToday()
    val services = filter.services(data)
    val members = filter.members(data)
    val families = filter.families(data)
    val up = upcomingServices(services, today)
    val birthdays = birthdaysToday(members, today)
    val nextBirthdays = upcomingBirthdays(members, today)
    val anniversaries = anniversariesToday(families, today)
    var selected by remember { mutableStateOf<Service?>(null) }
    // which accordions are open, kept on the device (survives tab switches and restarts)
    val context = LocalContext.current
    val prefs = remember { Prefs(context) }
    var open by remember { mutableStateOf(prefs.openSections) }
    fun toggle(key: String) {
        open = if (key in open) open - key else open + key
        prefs.openSections = open
    }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        update?.let { UpdateBanner(it) }
        if (error != null) {
            Text(
                "Tidak dapat terhubung ke server; menampilkan data tersimpan dari " +
                    DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT).format(Date(data.fetchedAt)) + ".",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        val notices = listOfNotNull(
            up.today.size.takeIf { it > 0 }?.let { "$it ibadah" },
            birthdays.size.takeIf { it > 0 }?.let { "$it ulang tahun" },
            anniversaries.size.takeIf { it > 0 }?.let { "$it HUT pernikahan" },
        ).joinToString(" · ").ifEmpty { "Tidak ada ibadah atau ulang tahun" } + " hari ini"
        Accordion("🔔 Pemberitahuan", notices, open = "notices" in open, onToggle = { toggle("notices") }) {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                SectionCard("⛪", "Ibadah Hari Ini & Minggu Ini", longDate(today) + (filter.type(data)?.let { " · Ibadah ${it.name} saja" } ?: "")) {
                    if (up.today.isEmpty()) Text("Tidak ada ibadah hari ini.", style = MaterialTheme.typography.bodyLarge)
                    // today's and this week's together, per kolom; today's show the time only
                    KolomGroups(up.today + up.week, { it.kolom }) { s ->
                        val isToday = s in up.today
                        ServiceRow(s, withDate = !isToday, highlight = isToday) { selected = s }
                    }
                    if (up.week.isEmpty() && up.next != null) {
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
                    // one line per kolom: "Budi Umboh, Erwin Umboh"
                    birthdays.byKolom { it.kolom }.forEach { (k, list) ->
                        Surface(shape = RoundedCornerShape(10.dp), color = MaterialTheme.colorScheme.secondaryContainer, modifier = Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp)) {
                                Text(
                                    "${kolomHeading(k)}:".uppercase(),
                                    style = MaterialTheme.typography.labelMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(list.joinToString(", ") { it.name }, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                            }
                        }
                    }
                    if (nextBirthdays.isNotEmpty()) {
                        Text("Berikutnya:", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        // per kolom, one line each: "Budi Umboh (besok), Sari Kowaas (25 Okt)"
                        nextBirthdays.byKolom { it.first.kolom }.forEach { (k, list) ->
                            Text(
                                "${kolomHeading(k)}:".uppercase(),
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Text(
                                list.joinToString(", ") { (m, days) ->
                                    "${m.name} (" + (if (days == 1L) "besok" else "${m.birthDay} ${MONTHS[m.birthMonth!! - 1].take(3)}") + ")"
                                },
                                style = MaterialTheme.typography.bodyLarge,
                            )
                        }
                    }
                }

                if (anniversaries.isNotEmpty()) {
                    SectionCard("💍", "HUT Pernikahan Hari Ini", accent = MaterialTheme.colorScheme.tertiary) {
                        anniversaries.forEach { Text("Kel. ${it.name}", style = MaterialTheme.typography.titleMedium) }
                    }
                }
            }
        }

        Accordion(
            "📊 Statistik",
            "${filter.label(data)} · ${families.size} keluarga · ${members.size} jiwa",
            open = "stats" in open,
            onToggle = { toggle("stats") },
        ) {
            StatsSection(data, filter, onCategory, onMembers)
        }
    }

    selected?.let { ServiceSheet(it) { selected = null } }
}

