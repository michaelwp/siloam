package id.gmimsiloam.jemaat.ui

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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.hasBirthday
import id.gmimsiloam.jemaat.data.inKolom
import java.time.LocalDate

/** Birthdays month by month (day and month only: the public view has no birth year). */
@Composable
fun BirthdaysScreen(data: Snapshot, kolom: Int?) {
    val today = remember { LocalDate.now() }
    var month by remember { mutableIntStateOf(today.monthValue) }
    val list = data.members.inKolom(kolom) { it.kolom }
        .filter { it.hasBirthday() && it.birthMonth == month }
        .sortedWith(compareBy({ it.birthDay }, { it.name.lowercase() }))

    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { month = if (month == 1) 12 else month - 1 }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowLeft, contentDescription = "Bulan sebelumnya")
            }
            Text(MONTHS[month - 1], style = MaterialTheme.typography.titleLarge, textAlign = TextAlign.Center, modifier = Modifier.weight(1f))
            IconButton(onClick = { month = if (month == 12) 1 else month + 1 }) {
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = "Bulan berikutnya")
            }
        }
        Text(
            "${list.size} anggota jemaat berulang tahun di bulan ${MONTHS[month - 1]}",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        if (list.isEmpty()) {
            Message("Tidak ada ulang tahun di bulan ini.")
        } else {
            LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(list, key = { it.id }) { m ->
                    val isToday = month == today.monthValue && m.birthDay == today.dayOfMonth
                    Surface(
                        shape = RoundedCornerShape(10.dp),
                        color = if (isToday) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surface,
                        tonalElevation = 1.dp,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.width(56.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                Text("${m.birthDay}", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                                Text(MONTHS[month - 1].take(3), style = MaterialTheme.typography.labelMedium)
                            }
                            Column(Modifier.weight(1f)) {
                                Text(m.name + if (isToday) " 🎂" else "", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                                Text("Kel. ${m.familyName} · Kolom ${m.kolom}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        }
                    }
                }
            }
        }
    }
}
