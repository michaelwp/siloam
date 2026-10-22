@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.provider.CalendarContract
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.WEEKDAYS
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.kolomText
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.monthGrid
import id.gmimsiloam.jemaat.data.timeRange
import id.gmimsiloam.jemaat.data.weekday
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.YearMonth

/**
 * The schedule month by month, as a list by date (like the web's "Jadwal"
 * view) or as a month calendar; the choice is remembered.
 */
@Composable
fun ScheduleScreen(data: Snapshot, filter: DataFilter) {
    val today = rememberToday()
    val context = LocalContext.current
    val prefs = remember { Prefs(context) }
    var calendar by remember { mutableStateOf(prefs.scheduleCalendar) }
    // kept across tab switches and rotation (SiloamApp keeps each tab's saved state)
    var month by rememberSaveable { mutableStateOf(YearMonth.from(today)) }
    var day by rememberSaveable { mutableStateOf(today) } // the calendar's chosen day
    var hidden by rememberSaveable { mutableStateOf(setOf<String>()) }
    var selected by remember { mutableStateOf<Service?>(null) }
    val shown = filter.services(data)
        .filter { it.typeId !in hidden }
        .sortedWith(compareBy({ it.date }, { it.time }))
    fun goTo(m: YearMonth) {
        month = m
        // keep the chosen day in the shown month: today there, else the 1st
        day = if (YearMonth.from(today) == m) today else m.atDay(1)
    }

    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { goTo(month.minusMonths(1)) }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowLeft, contentDescription = "Bulan sebelumnya")
            }
            Text(
                "${MONTHS[month.monthValue - 1]} ${month.year}",
                style = MaterialTheme.typography.titleLarge,
                textAlign = TextAlign.Center,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = { goTo(month.plusMonths(1)) }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = "Bulan berikutnya")
            }
            TextButton(onClick = { goTo(YearMonth.from(today)) }) { Text("Hari ini") }
            IconButton(onClick = {
                calendar = !calendar
                prefs.scheduleCalendar = calendar
            }) {
                if (calendar) {
                    Icon(Icons.AutoMirrored.Filled.List, contentDescription = "Tampilan daftar")
                } else {
                    Icon(Icons.Filled.DateRange, contentDescription = "Tampilan kalender")
                }
            }
        }
        LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(data.types, key = { it.id }) { t ->
                FilterChip(
                    selected = t.id !in hidden,
                    onClick = { hidden = if (t.id in hidden) hidden - t.id else hidden + t.id },
                    label = { Text(t.name) },
                    leadingIcon = { ColorDot(typeColor(t.color)) },
                )
            }
        }
        if (calendar) {
            MonthCalendar(
                month = month,
                services = shown,
                today = today,
                day = day,
                onDay = { d ->
                    // a day of the previous / next month opens that month
                    if (YearMonth.from(d) != month) month = YearMonth.from(d)
                    day = d
                },
                onMonth = ::goTo,
                onService = { selected = it },
            )
        } else {
            MonthList(shown.filter { YearMonth.from(it.date) == month }, today) { selected = it }
        }
    }
    selected?.let { ServiceSheet(it) { selected = null } }
}

@Composable
private fun MonthList(services: List<Service>, today: LocalDate, onService: (Service) -> Unit) {
    val byDay = services.groupBy { it.date }
    if (byDay.isEmpty()) {
        Message("Tidak ada jadwal ibadah di bulan ini.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
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
                        list.forEach { s -> ServiceRow(s, withDate = false, highlight = day == today) { onService(s) } }
                    }
                }
            }
        }
    }
}

/** At most this many services are drawn in a calendar day; the rest show as "+n". */
private const val CELL_SERVICES = 2

/**
 * A month grid: each day shows its services as time chips in the jenis color; tapping a day lists
 * its services below the grid. Swiping sideways changes the month.
 */
@Composable
private fun MonthCalendar(
    month: YearMonth,
    services: List<Service>,
    today: LocalDate,
    day: LocalDate,
    onDay: (LocalDate) -> Unit,
    onMonth: (YearMonth) -> Unit,
    onService: (Service) -> Unit,
) {
    val byDay = services.groupBy { it.date }
    val chosen = byDay[day].orEmpty()

    LazyColumn(contentPadding = PaddingValues(start = 8.dp, end = 8.dp, top = 8.dp, bottom = 24.dp)) {
        item(key = "grid") {
            MonthGrid(
                month = month,
                today = today,
                day = day,
                onDay = onDay,
                onMonth = onMonth,
                clickLabel = "Lihat jadwal",
                describe = { d -> byDay[d]?.let { ", ${it.size} ibadah" } ?: "" },
            ) { d, inMonth ->
                val list = byDay[d].orEmpty()
                list.take(CELL_SERVICES).forEach { s ->
                    Text(
                        clock(s.time),
                        style = MaterialTheme.typography.labelSmall,
                        color = onTypeColor(typeColor(s.typeColor)),
                        maxLines = 1,
                        textAlign = TextAlign.Center,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(4.dp))
                            .background(typeColor(s.typeColor).copy(alpha = if (inMonth) 1f else 0.45f))
                            .padding(vertical = 1.dp),
                    )
                }
                if (list.size > CELL_SERVICES) {
                    Text("+${list.size - CELL_SERVICES}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        item(key = "day") {
            Text(
                longDate(day),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(start = 8.dp, end = 8.dp, top = 16.dp, bottom = 8.dp),
            )
            if (chosen.isEmpty()) {
                Text(
                    "Tidak ada ibadah pada hari ini.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 8.dp),
                )
            }
        }
        items(chosen, key = { it.id }) { s ->
            Column(Modifier.padding(horizontal = 8.dp, vertical = 4.dp)) {
                ServiceRow(s, withDate = false, highlight = day == today) { onService(s) }
            }
        }
    }
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
                "Khadim" to s.leader,
                "Pembawa Acara" to s.mc,
                "Kolom" to if (s.kolom == 0) "Semua kolom" else "${s.kolom}",
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
                s.kolomText,
                s.leader.ifEmpty { null }?.let { "Khadim: $it" },
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
        if (s.leader.isNotEmpty()) appendLine("Khadim: ${s.leader}")
        if (s.mc.isNotEmpty()) append("Pembawa Acara: ${s.mc}")
    }.trim()
    context.startActivity(
        Intent.createChooser(Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, text), "Bagikan jadwal ibadah"),
    )
}
