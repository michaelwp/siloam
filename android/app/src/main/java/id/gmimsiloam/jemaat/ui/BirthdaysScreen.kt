package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Member
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.birthdaysOn
import id.gmimsiloam.jemaat.data.hasBirthday
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.monthGrid
import java.time.LocalDate
import java.time.YearMonth

/** The birthday color in the calendar (as on the web). */
private val BIRTHDAY = Color(0xFFD9468F)

/**
 * Birthdays month by month (day and month only: the public view has no birth year), as a list or
 * as a month calendar like the Jadwal tab's; the choice is remembered.
 */
@Composable
fun BirthdaysScreen(data: Snapshot, filter: DataFilter) {
    val today = rememberToday()
    val context = LocalContext.current
    val prefs = remember { Prefs(context) }
    var calendar by remember { mutableStateOf(prefs.birthdaysCalendar) }
    var month by rememberSaveable { mutableStateOf(YearMonth.from(today)) }
    var day by rememberSaveable { mutableStateOf(today) } // the calendar's chosen day
    val members = filter.members(data)
    fun goTo(m: YearMonth) {
        month = m
        day = if (YearMonth.from(today) == m) today else m.atDay(1)
    }

    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { goTo(month.minusMonths(1)) }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowLeft, contentDescription = "Bulan sebelumnya")
            }
            Text(
                // birthdays recur: the year only matters for the calendar's weekdays
                MONTHS[month.monthValue - 1] + if (calendar) " ${month.year}" else "",
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
                prefs.birthdaysCalendar = calendar
            }) {
                if (calendar) {
                    Icon(Icons.AutoMirrored.Filled.List, contentDescription = "Tampilan daftar")
                } else {
                    Icon(Icons.Filled.DateRange, contentDescription = "Tampilan kalender")
                }
            }
        }
        if (calendar) {
            BirthdayCalendar(
                members = members,
                month = month,
                today = today,
                day = day,
                onDay = { d ->
                    if (YearMonth.from(d) != month) month = YearMonth.from(d)
                    day = d
                },
                onMonth = ::goTo,
            )
        } else {
            BirthdayList(members, month.monthValue, today)
        }
    }
}

@Composable
private fun BirthdayList(members: List<Member>, month: Int, today: LocalDate) {
    val list = members
        .filter { it.hasBirthday() && it.birthMonth == month }
        .sortedWith(compareBy({ it.birthDay }, { it.name.lowercase() }))
    Column {
        Text(
            "${list.size} anggota jemaat berulang tahun di bulan ${MONTHS[month - 1]}",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        if (list.isEmpty()) {
            Message("Tidak ada ulang tahun di bulan ini.")
            return
        }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            items(list, key = { it.id }) { m ->
                BirthdayRow(m, isToday = month == today.monthValue && m.birthDay == today.dayOfMonth)
            }
        }
    }
}

/**
 * A month grid: each day shows how many have a birthday then; tapping a day lists them below the
 * grid. Swiping sideways changes the month.
 */
@Composable
private fun BirthdayCalendar(
    members: List<Member>,
    month: YearMonth,
    today: LocalDate,
    day: LocalDate,
    onDay: (LocalDate) -> Unit,
    onMonth: (YearMonth) -> Unit,
) {
    val byDay = remember(members, month) { birthdaysOn(members, monthGrid(month)) }
    val chosen = byDay[day].orEmpty()

    LazyColumn(contentPadding = PaddingValues(start = 8.dp, end = 8.dp, top = 8.dp, bottom = 24.dp)) {
        item(key = "grid") {
            MonthGrid(
                month = month,
                today = today,
                day = day,
                onDay = onDay,
                onMonth = onMonth,
                clickLabel = "Lihat ulang tahun",
                describe = { d -> byDay[d]?.let { ", ${it.size} ulang tahun" } ?: "" },
            ) { d, inMonth ->
                byDay[d]?.let { list ->
                    Text(
                        "🎂 ${list.size}",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.White,
                        maxLines = 1,
                        textAlign = TextAlign.Center,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(4.dp))
                            .background(BIRTHDAY.copy(alpha = if (inMonth) 1f else 0.45f))
                            .padding(vertical = 1.dp),
                    )
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
                    "Tidak ada ulang tahun pada hari ini.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 8.dp),
                )
            }
        }
        items(chosen, key = { it.id }) { m ->
            Column(Modifier.padding(horizontal = 8.dp, vertical = 4.dp)) {
                BirthdayRow(m, isToday = day == today)
            }
        }
    }
}

@Composable
private fun BirthdayRow(m: Member, isToday: Boolean) {
    Surface(
        shape = RoundedCornerShape(10.dp),
        color = if (isToday) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surface,
        tonalElevation = 1.dp,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.width(56.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Text("${m.birthDay}", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                Text(MONTHS[m.birthMonth!! - 1].take(3), style = MaterialTheme.typography.labelMedium)
            }
            Column(Modifier.weight(1f)) {
                Text(m.name + if (isToday) " 🎂" else "", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                Text("Kel. ${m.familyName} · Kolom ${m.kolom}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}
