@file:OptIn(ExperimentalLayoutApi::class)

package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.Category
import id.gmimsiloam.jemaat.data.MONTHS
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.inCategory
import id.gmimsiloam.jemaat.data.stats
import java.time.LocalDate

/**
 * The statistics shown on Beranda (in its Statistik accordion): totals, BIPRA counts (a tap opens that
 * category on the BIPRA tab) and charts. Laid out in the caller's scrolling column.
 */
@Composable
fun StatsSection(data: Snapshot, filter: DataFilter, onCategory: (String) -> Unit) {
    val members = filter.members(data)
    val families = filter.families(data)
    val s = stats(members, families, data.categories)
    val month = remember { LocalDate.now().monthValue }

    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        TileRow(listOf("Keluarga" to s.families, "Jiwa" to s.members))
        TileRow(listOf("Laki-laki" to s.male, "Perempuan" to s.female))

        Text("Kategorial BIPRA", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        FlowRow(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalArrangement = Arrangement.spacedBy(10.dp), maxItemsInEachRow = 3) {
            data.categories.forEach { c ->
                Tile(c.label, s.bipra[c.key] ?: 0, c.range, Modifier.weight(1f).clickable { onCategory(c.key) })
            }
        }
        if (s.uncategorized > 0) {
            Text(
                "${s.uncategorized} anggota belum memiliki tanggal lahir sehingga tidak berkategori.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        ChartCard("Ulang tahun per bulan", "Jumlah jiwa yang berulang tahun tiap bulan · ${MONTHS[month - 1]} disorot") {
            ColumnChart(
                data = MONTHS.mapIndexed { i, name -> ColumnDatum(name, name.take(3), s.birthdaysPerMonth[i], emphasis = i + 1 == month) },
                unit = "jiwa",
                labelHead = "Bulan",
                emphasize = true,
            )
        }
        ChartCard("Kategorial BIPRA per jenis kelamin", "Jumlah jiwa per kategori") {
            StackedBarChart(
                rows = s.bipraByGender.map { (c, g) -> StackRow(c.label, listOf(g.male, g.female)) },
                series = listOf(Series("Laki-laki", VizColors.series1()), Series("Perempuan", VizColors.series2())),
                rowHead = "Kategori",
            )
        }
        if (s.familySizes.isNotEmpty()) {
            ChartCard("Ukuran keluarga", "Jumlah keluarga menurut banyaknya anggota") {
                ColumnChart(
                    data = s.familySizes.mapIndexed { i, n -> ColumnDatum("${i + 1} jiwa", "${i + 1}", n) },
                    unit = "keluarga",
                    labelHead = "Anggota per keluarga",
                )
            }
        }
    }
}

@Composable
private fun TileRow(items: List<Pair<String, Int>>) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.fillMaxWidth()) {
        items.forEach { (label, value) -> Tile(label, value, null, Modifier.weight(1f)) }
    }
}

@Composable
private fun Tile(label: String, value: Int, range: String?, modifier: Modifier) {
    Surface(shape = RoundedCornerShape(12.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 1.dp, modifier = modifier) {
        Column(Modifier.padding(12.dp)) {
            Text("$value", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text(label, style = MaterialTheme.typography.bodyMedium)
            if (range != null) Text(range, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ChartCard(title: String, subtitle: String, content: @Composable () -> Unit) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        border = androidx.compose.foundation.BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            content()
        }
    }
}

/** The BIPRA tab: members per category (as the web's Kategorial BIPRA page), searchable. */
@Composable
fun BipraScreen(data: Snapshot, filter: DataFilter, category: String?, onCategory: (String?) -> Unit) {
    var q by rememberSaveable { mutableStateOf("") }
    val members = filter.members(data)
    val kolom = filter.kolomOf(data)
    val query = q.trim().lowercase()
    val all: List<Category?> = listOf(null) + data.categories
    val chosen = data.categories.firstOrNull { it.key == category }
    val rows = members
        .filter { chosen == null || inCategory(it, chosen) }
        .filter { query.isEmpty() || it.name.lowercase().contains(query) || it.familyName.lowercase().contains(query) }
        .sortedBy { it.name.lowercase() }

    // keep the chosen category's chip in view (e.g. after tapping a tile)
    val chips = rememberLazyListState()
    LaunchedEffect(category) {
        chips.animateScrollToItem(all.indexOfFirst { it?.key == category }.coerceAtLeast(0))
    }

    Column(Modifier.fillMaxSize().padding(top = 8.dp)) {
        LazyRow(state = chips, contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(all, key = { it?.key ?: "all" }) { c ->
                val n = if (c == null) members.size else members.count { inCategory(it, c) }
                FilterChip(
                    selected = category == c?.key,
                    onClick = { onCategory(if (category == c?.key) null else c?.key) },
                    label = { Text("${c?.label ?: "Semua"} · $n") },
                )
            }
        }
        OutlinedTextField(
            value = q,
            onValueChange = { q = it },
            leadingIcon = { Icon(Icons.Filled.Search, contentDescription = null) },
            placeholder = { Text("Cari nama / keluarga…") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        )
        Text(
            "${rows.size} jiwa" + (chosen?.let { " · ${it.label} (${it.range})" } ?: ""),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        if (rows.isEmpty()) {
            Message("Tidak ada anggota yang cocok.")
            return
        }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            items(rows, key = { it.id }) { m ->
                Surface(shape = RoundedCornerShape(10.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 1.dp, modifier = Modifier.fillMaxWidth()) {
                    Row(Modifier.padding(12.dp)) {
                        Column(Modifier.weight(1f)) {
                            Text(m.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                            Text(
                                listOfNotNull(
                                    "Kel. ${m.familyName}",
                                    if (kolom == null) "Kolom ${m.kolom}" else null,
                                    m.gender.ifEmpty { null },
                                ).joinToString(" · "),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        Column(Modifier.width(96.dp), horizontalAlignment = androidx.compose.ui.Alignment.End, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Pill(data.categories.firstOrNull { it.key == m.category }?.label ?: m.category.ifEmpty { "Tanpa kategori" })
                            m.flags.forEach { f ->
                                Pill(data.categories.firstOrNull { it.key == f }?.label ?: f, MaterialTheme.colorScheme.primaryContainer)
                            }
                        }
                    }
                }
            }
        }
    }
}
