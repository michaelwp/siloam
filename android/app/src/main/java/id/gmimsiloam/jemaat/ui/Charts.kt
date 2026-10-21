package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.ceil
import kotlin.math.floor
import kotlin.math.log10
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

// Small charts following the web's dataviz rules: thin marks (<= 24dp) with a
// 4dp rounded data end and a square baseline, 2dp gaps between stacked
// segments, a hairline grid, text in text colors (never the series color),
// a value on tap, and a table view so no number lives only in the chart.

/** The palette validated for the web app (light / dark surfaces). */
object VizColors {
    @Composable
    fun series1() = if (isSystemInDarkTheme()) Color(0xFF3987E5) else Color(0xFF2A78D6)

    @Composable
    fun series2() = if (isSystemInDarkTheme()) Color(0xFFD95926) else Color(0xFFEB6834)

    @Composable
    fun deemph() = if (isSystemInDarkTheme()) Color(0xFF4A4A46) else Color(0xFFC3C2B7)
}

private const val RADIUS_DP = 4f
private const val BAR_MAX_DP = 24f

/** Clean axis ticks (steps of 1, 2 or 5 × 10^n) covering 0..max. */
private fun niceTicks(maxValue: Int, target: Int = 4): Pair<List<Int>, Int> {
    val raw = max(maxValue, 1).toDouble() / target
    val pow = 10.0.pow(floor(log10(raw)))
    val step = max(1, (listOf(1.0, 2.0, 5.0, 10.0).map { it * pow }.first { it >= raw }).toInt())
    val top = max(step, (ceil(maxValue.toDouble() / step) * step).toInt())
    return (0..top step step).toList() to top
}

/** Column: rounded top (the data end), square at the baseline. */
private fun DrawScope.column(x: Float, top: Float, w: Float, bottom: Float, color: Color) {
    val r = min(RADIUS_DP.dp.toPx(), min(w / 2, bottom - top))
    val path = Path().apply {
        addRoundRect(RoundRect(x, top, x + w, bottom, CornerRadius(r), CornerRadius(r), CornerRadius.Zero, CornerRadius.Zero))
    }
    drawPath(path, color)
}

/** Horizontal bar end: rounded on the right (data end), square on the left. */
private fun DrawScope.barEnd(x: Float, top: Float, w: Float, h: Float, color: Color) {
    val r = min(RADIUS_DP.dp.toPx(), min(h / 2, w))
    val path = Path().apply {
        addRoundRect(RoundRect(x, top, x + w, top + h, CornerRadius.Zero, CornerRadius(r), CornerRadius(r), CornerRadius.Zero))
    }
    drawPath(path, color)
}

data class ColumnDatum(val label: String, val short: String, val value: Int, val emphasis: Boolean = false)

/**
 * Vertical columns for one series. With [emphasize], emphasized columns are in
 * the accent and the rest de-emphasized gray; only those (or, without
 * emphasis, the largest) carry a value label. Tap a column for its value.
 */
@Composable
fun ColumnChart(data: List<ColumnDatum>, unit: String, labelHead: String, emphasize: Boolean = false) {
    val accent = VizColors.series1()
    val deemph = VizColors.deemph()
    val grid = MaterialTheme.colorScheme.outlineVariant
    val ink = MaterialTheme.colorScheme.onSurface
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val measurer = rememberTextMeasurer()
    var selected by remember(data) { mutableStateOf<Int?>(null) }
    val maxValue = data.maxOfOrNull { it.value } ?: 0
    val (ticks, top) = niceTicks(maxValue)
    fun labelled(d: ColumnDatum) = d.value > 0 && if (emphasize) d.emphasis else d.value == maxValue

    Column {
        Text(
            selected?.let { "${data[it].label}: ${data[it].value} $unit" } ?: "Ketuk batang untuk melihat angka",
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (selected != null) FontWeight.SemiBold else FontWeight.Normal,
            color = if (selected != null) ink else muted,
        )
        Canvas(
            Modifier
                .fillMaxWidth()
                .height(180.dp)
                .padding(top = 6.dp)
                .semantics { contentDescription = "$labelHead: " + data.joinToString(", ") { "${it.label} ${it.value}" } }
                .pointerInput(data) {
                    detectTapGestures { pos ->
                        val left = 28.dp.toPx()
                        val band = (size.width - left) / max(data.size, 1)
                        val i = ((pos.x - left) / band).toInt()
                        selected = if (i in data.indices && pos.x >= left) (if (selected == i) null else i) else null
                    }
                },
        ) {
            val left = 28.dp.toPx()
            val bottomPad = 22.dp.toPx()
            val topPad = 16.dp.toPx()
            val plotH = size.height - topPad - bottomPad
            val y = { v: Int -> topPad + plotH - v.toFloat() / top * plotH }
            val tickStyle = TextStyle(color = muted, fontSize = 10.sp)
            // hairline grid with tick labels; the baseline slightly stronger
            ticks.forEach { t ->
                drawLine(if (t == 0) muted else grid, Offset(left, y(t)), Offset(size.width, y(t)), strokeWidth = 1.dp.toPx() / if (t == 0) 1f else 2f)
                val lay = measurer.measure("$t", tickStyle)
                drawText(lay, topLeft = Offset(left - 6.dp.toPx() - lay.size.width, y(t) - lay.size.height / 2))
            }
            val band = (size.width - left) / max(data.size, 1)
            val barW = min(BAR_MAX_DP.dp.toPx(), band * 0.62f)
            data.forEachIndexed { i, d ->
                val cx = left + band * i + band / 2
                if (d.value > 0) {
                    val color = if (emphasize && !d.emphasis) deemph else accent
                    column(cx - barW / 2, y(d.value), barW, y(0), color)
                }
                if (selected == i) {
                    drawRect(grid.copy(alpha = 0.5f), Offset(left + band * i, topPad), Size(band, plotH))
                }
                if (labelled(d) || selected == i) {
                    val lay = measurer.measure("${d.value}", TextStyle(color = ink, fontSize = 11.sp, fontWeight = FontWeight.SemiBold))
                    drawText(lay, topLeft = Offset(cx - lay.size.width / 2, y(d.value) - lay.size.height - 2.dp.toPx()))
                }
                val lay = measurer.measure(d.short, tickStyle)
                drawText(lay, topLeft = Offset(cx - lay.size.width / 2, size.height - lay.size.height))
            }
        }
        TableView(listOf(labelHead, unit), data.map { listOf(it.label, "${it.value}") })
    }
}

data class StackRow(val label: String, val values: List<Int>)
data class Series(val name: String, val color: Color)

/**
 * Horizontal stacked bars (part-to-whole per row) with a legend, a total at
 * each bar end and 2dp gaps between segments. Tap a row for its numbers.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun StackedBarChart(rows: List<StackRow>, series: List<Series>, rowHead: String) {
    val ink = MaterialTheme.colorScheme.onSurface
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val measurer = rememberTextMeasurer()
    var selected by remember(rows) { mutableStateOf<Int?>(null) }
    val maxTotal = max(1, rows.maxOfOrNull { it.values.sum() } ?: 0)
    val rowH = 36.dp

    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        // legend: identity never by color alone
        FlowRow(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            series.forEach { s ->
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    ColorDot(s.color)
                    Text(s.name, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
        Text(
            selected?.let { i ->
                val r = rows[i]
                "${r.label}: " + series.mapIndexed { k, s -> "${r.values[k]} ${s.name.lowercase()}" }.joinToString(", ") + " (${r.values.sum()} jiwa)"
            } ?: "Ketuk baris untuk melihat angka",
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (selected != null) FontWeight.SemiBold else FontWeight.Normal,
            color = if (selected != null) ink else muted,
        )
        Canvas(
            Modifier
                .fillMaxWidth()
                .height(rowH * rows.size)
                .semantics {
                    contentDescription = "$rowHead per " + series.joinToString(" dan ") { it.name } + ": " +
                        rows.joinToString("; ") { r -> r.label + " " + r.values.joinToString(" / ") }
                }
                .pointerInput(rows) {
                    detectTapGestures { pos ->
                        val i = (pos.y / rowH.toPx()).toInt()
                        selected = if (i in rows.indices) (if (selected == i) null else i) else null
                    }
                },
        ) {
            val labelW = 92.dp.toPx()
            val totalW = 34.dp.toPx()
            val plotW = size.width - labelW - totalW
            val barH = min(BAR_MAX_DP.dp.toPx(), rowH.toPx() - 8.dp.toPx())
            val gap = 2.dp.toPx()
            rows.forEachIndexed { i, r ->
                val top = i * rowH.toPx() + (rowH.toPx() - barH) / 2
                val label = measurer.measure(r.label, TextStyle(color = ink, fontSize = 12.sp))
                drawText(label, topLeft = Offset(labelW - 8.dp.toPx() - label.size.width, top + (barH - label.size.height) / 2))
                val last = r.values.indexOfLast { it > 0 }
                var x = labelW
                r.values.forEachIndexed { k, v ->
                    if (v <= 0) return@forEachIndexed
                    val w = v.toFloat() / maxTotal * plotW
                    if (k == last) barEnd(x, top, w, barH, series[k].color)
                    else drawRect(series[k].color, Offset(x, top), Size(max(w - gap, 0f), barH))
                    x += w
                }
                val total = measurer.measure("${r.values.sum()}", TextStyle(color = ink, fontSize = 12.sp, fontWeight = FontWeight.SemiBold))
                drawText(total, topLeft = Offset(x + 6.dp.toPx(), top + (barH - total.size.height) / 2))
                if (selected == i) {
                    drawRect(muted.copy(alpha = 0.12f), Offset(0f, i * rowH.toPx()), Size(size.width, rowH.toPx()))
                }
            }
        }
        TableView(
            listOf(rowHead) + series.map { it.name } + "Jumlah",
            rows.map { r -> listOf(r.label) + r.values.map { "$it" } + "${r.values.sum()}" },
        )
    }
}

/** Collapsible table twin of a chart. */
@Composable
private fun TableView(head: List<String>, rows: List<List<String>>) {
    var open by remember { mutableStateOf(false) }
    TextButton(onClick = { open = !open }, contentPadding = androidx.compose.foundation.layout.PaddingValues(0.dp)) {
        Text(if (open) "Sembunyikan tabel" else "Lihat tabel")
    }
    if (!open) return
    Column {
        TableRow(head, bold = true)
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        rows.forEach { TableRow(it, bold = false) }
    }
}

@Composable
private fun TableRow(cells: List<String>, bold: Boolean) {
    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        cells.forEachIndexed { i, c ->
            Text(
                c,
                style = MaterialTheme.typography.bodySmall,
                fontWeight = if (bold) FontWeight.SemiBold else FontWeight.Normal,
                textAlign = if (i == 0) androidx.compose.ui.text.style.TextAlign.Start else androidx.compose.ui.text.style.TextAlign.End,
                modifier = Modifier.weight(if (i == 0) 1.6f else 1f),
            )
        }
    }
}
