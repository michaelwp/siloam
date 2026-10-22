package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AssistChip
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.GENDER_FEMALE
import id.gmimsiloam.jemaat.data.GENDER_MALE
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Member
import id.gmimsiloam.jemaat.data.MemberFilter
import id.gmimsiloam.jemaat.data.Tri
import id.gmimsiloam.jemaat.data.maritalStatuses
import id.gmimsiloam.jemaat.data.Snapshot

private fun Boolean?.label(yes: String) = when (this) {
    true -> yes
    false -> "Belum ${yes.lowercase()}"
    null -> null
}

/**
 * Families and their members, searchable by family or member name, and filtered by baptis, sidi,
 * marital status and gender ([memberFilter]; then only the matching members, and the families they are in).
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun MembersScreen(data: Snapshot, filter: DataFilter, memberFilter: MemberFilter, opened: Int, onMemberFilter: (MemberFilter) -> Unit) {
    var q by rememberSaveable { mutableStateOf("") }
    // a new [opened] (the tab opened from a Beranda tile) starts with an empty search; kept with the
    // search, as the tab's saved state comes back when it is shown again
    var searchOf by rememberSaveable { mutableIntStateOf(opened) }
    if (searchOf != opened) {
        searchOf = opened
        q = ""
    }
    val query = q.trim().lowercase()
    // a BIPRA filter: its members only, and the families they are in
    val all = filter.members(data)
    val byFamily = all.filter(memberFilter::matches).groupBy { it.familyId }
    val kolom = filter.kolomOf(data)
    val families = filter.families(data)
        .sortedWith(compareBy({ it.kolom }, { it.no }))
        .map { f -> f to (byFamily[f.id] ?: emptyList()) }
        .filter { (_, ms) -> !memberFilter.active || ms.isNotEmpty() }
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
        FilterRow(memberFilter, maritalStatuses(all), onMemberFilter)
        Text(
            "${families.size} keluarga · ${families.sumOf { it.second.size }} jiwa",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        if (families.isEmpty()) {
            Message(
                when {
                    query.isNotEmpty() -> "Tidak ada yang cocok dengan \"$q\"."
                    memberFilter.active -> "Tidak ada anggota yang cocok dengan filter."
                    else -> "Belum ada data keluarga."
                },
            )
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
                            // filtered: the matching members of the whole family
                            if (ms.size < f.memberCount) "${ms.size} dari ${f.memberCount} jiwa" else "${ms.size} jiwa",
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

/** Baptis, Sidi, Status and Jenis kelamin chips; each opens its choices. "Hapus filter" (first) when any is set. */
@Composable
private fun FilterRow(f: MemberFilter, statuses: List<String>, onChange: (MemberFilter) -> Unit) {
    val tri = listOf(Tri.ANY to "Semua", Tri.YES to "Sudah", Tri.NO to "Belum", Tri.UNKNOWN to "Tanpa data")
    LazyRow(
        contentPadding = PaddingValues(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.padding(bottom = 4.dp),
    ) {
        // first, so it is in view while filters are set
        if (f.active) {
            item {
                AssistChip(
                    onClick = { onChange(MemberFilter.none) },
                    label = { Text("Hapus filter") },
                    leadingIcon = { Icon(Icons.Filled.Close, contentDescription = null, Modifier.size(18.dp)) },
                )
            }
        }
        item {
            ChoiceChip("Baptis", tri.first { it.first == f.baptized }.second.takeIf { f.baptized != Tri.ANY }, tri.map { it.second }) { i ->
                onChange(f.copy(baptized = tri[i].first))
            }
        }
        item {
            ChoiceChip("Sidi", tri.first { it.first == f.confirmed }.second.takeIf { f.confirmed != Tri.ANY }, tri.map { it.second }) { i ->
                onChange(f.copy(confirmed = tri[i].first))
            }
        }
        item {
            ChoiceChip("Status", f.status.ifEmpty { null }, listOf("Semua") + statuses) { i ->
                onChange(f.copy(status = if (i == 0) "" else statuses[i - 1]))
            }
        }
        item {
            val genders = listOf(GENDER_MALE, GENDER_FEMALE)
            ChoiceChip("Jenis kelamin", f.gender.ifEmpty { null }, listOf("Semua") + genders) { i ->
                onChange(f.copy(gender = if (i == 0) "" else genders[i - 1]))
            }
        }
    }
}

/** A filter chip "[name]: [value]" (selected) or "[name]" that opens [options]; [onPick] gets the index. */
@Composable
private fun ChoiceChip(name: String, value: String?, options: List<String>, onPick: (Int) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box {
        FilterChip(
            selected = value != null,
            onClick = { open = true },
            label = { Text(if (value == null) name else "$name: $value") },
            trailingIcon = { Icon(Icons.Filled.ArrowDropDown, contentDescription = null, Modifier.size(18.dp)) },
        )
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            options.forEachIndexed { i, o ->
                DropdownMenuItem(text = { Text(o) }, onClick = { onPick(i); open = false })
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
                *m.flags.toTypedArray(),
                m.maritalStatus.ifEmpty { null },
                m.baptized.label("Baptis"),
                m.confirmed.label("Sidi"),
                if (m.birthDay != null && m.birthMonth != null) "🎂 ${m.birthDay} ${MONTHS[m.birthMonth - 1].take(3)}" else null,
            ).forEach { Pill(it) }
        }
    }
}
