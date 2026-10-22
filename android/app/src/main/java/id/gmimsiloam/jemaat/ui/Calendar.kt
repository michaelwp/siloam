package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import id.gmimsiloam.jemaat.data.WEEKDAYS
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.monthGrid
import java.time.LocalDate
import java.time.YearMonth

/**
 * A month grid (like Google Calendar's month view), shared by the Jadwal and HUT tabs: each day shows
 * its number and [content] (e.g. service times); tapping a day calls [onDay]. Swiping sideways changes
 * the month. [describe] adds to a day's spoken label, e.g. ", 2 ibadah".
 */
@Composable
fun MonthGrid(
    month: YearMonth,
    today: LocalDate,
    day: LocalDate,
    onDay: (LocalDate) -> Unit,
    onMonth: (YearMonth) -> Unit,
    clickLabel: String,
    describe: (LocalDate) -> String,
    content: @Composable ColumnScope.(date: LocalDate, inMonth: Boolean) -> Unit,
) {
    val grid = remember(month) { monthGrid(month) }
    var drag by remember { mutableFloatStateOf(0f) }
    Column(
        Modifier.pointerInput(month) {
            detectHorizontalDragGestures(
                onDragStart = { drag = 0f },
                onDragEnd = {
                    val threshold = 64.dp.toPx()
                    if (drag > threshold) onMonth(month.minusMonths(1)) else if (drag < -threshold) onMonth(month.plusMonths(1))
                },
            ) { _, dx -> drag += dx }
        },
    ) {
        Row(Modifier.fillMaxWidth()) {
            WEEKDAYS.forEach {
                Text(
                    it.take(3),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.weight(1f).padding(vertical = 4.dp),
                )
            }
        }
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        grid.chunked(7).forEach { week ->
            Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min)) {
                week.forEach { d ->
                    val inMonth = YearMonth.from(d) == month
                    DayCell(
                        date = d,
                        inMonth = inMonth,
                        isToday = d == today,
                        isChosen = d == day,
                        clickLabel = clickLabel,
                        description = longDate(d) + describe(d),
                        onClick = { onDay(d) },
                        modifier = Modifier.weight(1f).fillMaxHeight(),
                    ) { content(d, inMonth) }
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        }
    }
}

@Composable
private fun DayCell(
    date: LocalDate,
    inMonth: Boolean,
    isToday: Boolean,
    isChosen: Boolean,
    clickLabel: String,
    description: String,
    onClick: () -> Unit,
    modifier: Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    val scheme = MaterialTheme.colorScheme
    Column(
        modifier
            .heightIn(min = 72.dp)
            .background(if (isChosen) scheme.primaryContainer.copy(alpha = 0.5f) else Color.Transparent)
            .clickable(onClickLabel = clickLabel, onClick = onClick)
            .semantics(mergeDescendants = true) { contentDescription = description }
            .padding(2.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Box(
            Modifier.size(26.dp).clip(CircleShape).background(if (isToday) scheme.primary else Color.Transparent),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                "${date.dayOfMonth}",
                style = MaterialTheme.typography.labelLarge,
                fontWeight = if (isToday) FontWeight.Bold else FontWeight.Normal,
                color = when {
                    isToday -> scheme.onPrimary
                    inMonth -> scheme.onSurface
                    else -> scheme.onSurfaceVariant.copy(alpha = 0.5f)
                },
            )
        }
        content()
    }
}
