package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Member
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.inKolom

private fun Boolean?.label(yes: String) = when (this) {
    true -> yes
    false -> "Belum ${yes.lowercase()}"
    null -> null
}

/** Families and their members, searchable by family or member name. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun MembersScreen(data: Snapshot, kolom: Int?) {
    var q by rememberSaveable { mutableStateOf("") }
    val query = q.trim().lowercase()
    val byFamily = data.members.groupBy { it.familyId }
    val families = data.families.inKolom(kolom) { it.kolom }
        .sortedWith(compareBy({ it.kolom }, { it.no }))
        .map { f -> f to (byFamily[f.id] ?: emptyList()) }
        .filter { (f, ms) -> query.isEmpty() || f.name.lowercase().contains(query) || ms.any { it.name.lowercase().contains(query) } }

    Column(Modifier.fillMaxSize()) {
        OutlinedTextField(
            value = q,
            onValueChange = { q = it },
            leadingIcon = { Icon(Icons.Filled.Search, contentDescription = null) },
            placeholder = { Text("Cari keluarga / anggota…") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        )
        Text(
            "${families.size} keluarga · ${families.sumOf { it.second.size }} jiwa",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        if (families.isEmpty()) {
            Message(if (query.isEmpty()) "Belum ada data keluarga." else "Tidak ada yang cocok dengan \"$q\".")
            return
        }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(families, key = { it.first.id }) { (f, ms) ->
                Surface(shape = RoundedCornerShape(12.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 1.dp, modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(
                            "${f.no}. Kel. ${f.name}",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                        )
                        val sub = listOfNotNull(
                            if (kolom == null) "Kolom ${f.kolom}" else null,
                            "${ms.size} jiwa",
                            f.note.ifEmpty { null },
                            if (f.weddingDay != null && f.weddingMonth != null) "HUT Pernikahan ${f.weddingDay} ${MONTHS[f.weddingMonth - 1]}" else null,
                        )
                        Text(sub.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        ms.forEachIndexed { i, m ->
                            if (i > 0) HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                            MemberLine(m, highlight = query.isNotEmpty() && m.name.lowercase().contains(query))
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun MemberLine(m: Member, highlight: Boolean) {
    Column(Modifier.padding(vertical = 4.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(
            m.name,
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = if (highlight) FontWeight.Bold else FontWeight.Medium,
            color = if (highlight) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
        )
        FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            listOfNotNull(
                m.category.ifEmpty { null },
                if (m.elderly) "Lansia" else null,
                m.maritalStatus.ifEmpty { null },
                m.baptized.label("Baptis"),
                m.confirmed.label("Sidi"),
                if (m.birthDay != null && m.birthMonth != null) "🎂 ${m.birthDay} ${MONTHS[m.birthMonth - 1].take(3)}" else null,
            ).forEach { Pill(it) }
        }
    }
}
