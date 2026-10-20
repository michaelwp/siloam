@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.provider.CalendarContract
import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.inKolom
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.timeRange
import id.gmimsiloam.jemaat.data.weekday
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.YearMonth

/** The schedule as a list by date (like the web's "Jadwal" view), month by month. */
@Composable
fun ScheduleScreen(data: Snapshot, kolom: Int?) {
    val today = remember { LocalDate.now() }
    var month by remember { mutableStateOf(YearMonth.from(today)) }
    var hidden by remember { mutableStateOf(setOf<String>()) }
    var selected by remember { mutableStateOf<Service?>(null) }
    val services = data.services.inKolom(kolom) { it.kolom }
        .filter { YearMonth.from(it.date) == month && it.typeId !in hidden }
        .sortedWith(compareBy({ it.date }, { it.time }))
    val byDay = services.groupBy { it.date }

    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { month = month.minusMonths(1) }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowLeft, contentDescription = "Bulan sebelumnya")
            }
            Text(
                "${MONTHS[month.monthValue - 1]} ${month.year}",
                style = MaterialTheme.typography.titleLarge,
                textAlign = TextAlign.Center,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = { month = month.plusMonths(1) }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = "Bulan berikutnya")
            }
            TextButton(onClick = { month = YearMonth.from(today) }) { Text("Hari ini") }
        }
        LazyRow(contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(data.types, key = { it.id }) { t ->
                FilterChip(
                    selected = t.id !in hidden,
                    onClick = { hidden = if (t.id in hidden) hidden - t.id else hidden + t.id },
                    label = { Text(t.name) },
                    leadingIcon = { ColorDot(typeColor(t.color)) },
                )
            }
        }
        if (byDay.isEmpty()) {
            Message("Tidak ada jadwal ibadah di bulan ini.")
        } else {
            LazyColumn(
                contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                byDay.forEach { (day, list) ->
                    item(key = day.toString()) {
                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            Column(Modifier.width(52.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                Text(
                                    "${day.dayOfMonth}",
                                    style = MaterialTheme.typography.headlineSmall,
                                    fontWeight = if (day == today) FontWeight.Bold else FontWeight.Normal,
                                    color = if (day == today) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                                )
                                Text(weekday(day).take(3).uppercase(), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                list.forEach { s -> ServiceRow(s, withDate = false, highlight = day == today) { selected = s } }
                            }
                        }
                    }
                }
            }
        }
    }
    selected?.let { ServiceSheet(it) { selected = null } }
}

/** Details of one service, with "add to my calendar" and "share". */
@Composable
fun ServiceSheet(s: Service, onDismiss: () -> Unit) {
    val context = LocalContext.current
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(start = 24.dp, end = 24.dp, bottom = 32.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                ColorDot(typeColor(s.typeColor), size = 14)
                Text("Ibadah ${s.typeName}", style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(s.title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
            Text("${longDate(s.date)} · ${timeRange(s.time)} WITA", style = MaterialTheme.typography.bodyLarge)
            listOf(
                "Lokasi" to s.place,
                "Pemimpin Ibadah" to s.leader,
                "Pembawa Acara" to s.mc,
                "Kolom" to "${s.kolom}",
                "Keterangan" to s.note,
            ).filter { it.second.isNotEmpty() }.forEach { (k, v) ->
                Row {
                    Text(k, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.width(130.dp))
                    Text(v, style = MaterialTheme.typography.bodyMedium)
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 8.dp)) {
                Button(onClick = { addToCalendar(context, s) }) {
                    Icon(Icons.Filled.DateRange, contentDescription = null)
                    Text("Tambah ke Kalender", modifier = Modifier.padding(start = 8.dp))
                }
                OutlinedButton(onClick = { share(context, s) }) {
                    Icon(Icons.Filled.Share, contentDescription = null)
                    Text("Bagikan", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}

private fun epochMillis(iso: String) = runCatching { OffsetDateTime.parse(iso).toInstant().toEpochMilli() }.getOrNull()

/** Opens the phone's calendar app (Google Calendar, Samsung Calendar, …) with the service filled in. */
private fun addToCalendar(context: Context, s: Service) {
    val intent = Intent(Intent.ACTION_INSERT)
        .setData(CalendarContract.Events.CONTENT_URI)
        .putExtra(CalendarContract.Events.TITLE, s.fullTitle)
        .putExtra(CalendarContract.Events.EVENT_LOCATION, s.place)
        .putExtra(
            CalendarContract.Events.DESCRIPTION,
            listOfNotNull(
                "Kolom ${s.kolom}",
                s.leader.ifEmpty { null }?.let { "Pemimpin Ibadah: $it" },
                s.mc.ifEmpty { null }?.let { "Pembawa Acara: $it" },
                s.note.ifEmpty { null },
            ).joinToString("\n"),
        )
    epochMillis(s.start)?.let { intent.putExtra(CalendarContract.EXTRA_EVENT_BEGIN_TIME, it) }
    epochMillis(s.end)?.let { intent.putExtra(CalendarContract.EXTRA_EVENT_END_TIME, it) }
    try {
        context.startActivity(intent)
    } catch (_: ActivityNotFoundException) {
        Toast.makeText(context, "Tidak ada aplikasi kalender di perangkat ini", Toast.LENGTH_LONG).show()
    }
}

private fun share(context: Context, s: Service) {
    val text = buildString {
        appendLine(s.fullTitle)
        appendLine("${longDate(s.date)}, ${timeRange(s.time)} WITA")
        if (s.place.isNotEmpty()) appendLine(s.place)
        if (s.leader.isNotEmpty()) appendLine("Pemimpin Ibadah: ${s.leader}")
        if (s.mc.isNotEmpty()) append("Pembawa Acara: ${s.mc}")
    }.trim()
    context.startActivity(
        Intent.createChooser(Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, text), "Bagikan jadwal ibadah"),
    )
}
