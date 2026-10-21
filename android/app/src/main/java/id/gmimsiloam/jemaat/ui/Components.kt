package id.gmimsiloam.jemaat.ui

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.data.AppRelease
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.byKolom
import id.gmimsiloam.jemaat.data.kolomHeading
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.shortDate
import kotlinx.coroutines.delay
import java.time.Duration
import java.time.LocalDate
import java.time.LocalDateTime

/** Centered message (loading, empty, error); scrollable so pull-to-refresh works. */
@Composable
fun Message(text: String) {
    Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(32.dp), contentAlignment = Alignment.Center) {
        Text(text, textAlign = TextAlign.Center, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

/**
 * Today's date, kept current: re-read whenever the app comes to the foreground
 * (Android keeps it in memory overnight) and at midnight while it is open.
 */
@Composable
fun rememberToday(): LocalDate {
    var today by remember { mutableStateOf(LocalDate.now()) }
    val owner = LocalLifecycleOwner.current
    LaunchedEffect(owner) {
        owner.repeatOnLifecycle(Lifecycle.State.STARTED) {
            while (true) {
                today = LocalDate.now()
                val midnight = LocalDate.now().plusDays(1).atStartOfDay()
                delay(Duration.between(LocalDateTime.now(), midnight).toMillis() + 1_000)
            }
        }
    }
    return today
}

/**
 * A collapsible container: a header (title and a one-line [summary], so a
 * closed section still tells something) that opens and closes [content].
 */
@Composable
fun Accordion(
    title: String,
    summary: String?,
    open: Boolean,
    onToggle: () -> Unit,
    content: @Composable () -> Unit,
) {
    val turn by animateFloatAsState(if (open) 180f else 0f, label = "chevron")
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        border = androidx.compose.foundation.BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(onClickLabel = if (open) "Tutup" else "Buka", onClick = onToggle)
                    .semantics { stateDescription = if (open) "Terbuka" else "Tertutup" }
                    .padding(horizontal = 16.dp, vertical = 14.dp),
            ) {
                Column(Modifier.weight(1f)) {
                    Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                    if (summary != null) {
                        Text(summary, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                Icon(Icons.Filled.KeyboardArrowDown, contentDescription = null, modifier = Modifier.rotate(turn))
            }
            AnimatedVisibility(visible = open, enter = expandVertically() + fadeIn(), exit = shrinkVertically() + fadeOut()) {
                Column(Modifier.padding(start = 12.dp, end = 12.dp, bottom = 12.dp)) { content() }
            }
        }
    }
}

/** Opens [url] (an https download link) in the browser. */
fun openLink(context: Context, url: String) {
    try {
        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
    } catch (_: ActivityNotFoundException) {
        Toast.makeText(context, "Tidak ada aplikasi untuk membuka tautan", Toast.LENGTH_LONG).show()
    }
}

/** "A new version is available" with a download button (Beranda and Pengaturan). */
@Composable
fun UpdateBanner(update: AppRelease) {
    val context = LocalContext.current
    Surface(shape = RoundedCornerShape(12.dp), color = MaterialTheme.colorScheme.primaryContainer, modifier = Modifier.fillMaxWidth()) {
        Row(Modifier.padding(start = 16.dp, end = 8.dp, top = 8.dp, bottom = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("Versi baru tersedia", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
                Text(
                    "Versi ${update.latestVersion} · terpasang ${BuildConfig.VERSION_NAME}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onPrimaryContainer,
                )
            }
            Button(onClick = { openLink(context, update.downloadUrl) }) { Text("Unduh") }
        }
    }
}

/**
 * Items per kolom, "Semua kolom" first: a small heading before each kolom's
 * items when there are several koloms (e.g. "Semua Kolom" is shown).
 */
@Composable
fun <T> KolomGroups(items: List<T>, kolomOf: (T) -> Int, item: @Composable (T) -> Unit) {
    val groups = items.byKolom(kolomOf)
    groups.forEach { (k, list) ->
        if (groups.size > 1) {
            Text(
                kolomHeading(k).uppercase(),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 4.dp),
            )
        }
        list.forEach { item(it) }
    }
}

/** A card with an emoji, a title and a subtitle. */
@Composable
fun SectionCard(
    emoji: String,
    title: String,
    subtitle: String? = null,
    accent: Color = MaterialTheme.colorScheme.primary,
    content: @Composable () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        border = androidx.compose.foundation.BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
    ) {
        Row(Modifier.height(IntrinsicSize.Min)) {
            Box(Modifier.width(4.dp).fillMaxHeight().background(accent))
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(emoji, style = MaterialTheme.typography.headlineMedium)
                    Column(Modifier.padding(start = 12.dp)) {
                        Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                        if (subtitle != null) {
                            Text(subtitle, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
                content()
            }
        }
    }
}

/** One service: a stripe in its jenis color, time, title and leaders. */
@Composable
fun ServiceRow(s: Service, withDate: Boolean, highlight: Boolean = false, onClick: (() -> Unit)? = null) {
    val color = typeColor(s.typeColor)
    Surface(
        shape = RoundedCornerShape(10.dp),
        color = if (highlight) color.copy(alpha = 0.14f) else MaterialTheme.colorScheme.surfaceVariant,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
    ) {
        Row(Modifier.height(IntrinsicSize.Min)) {
            Box(Modifier.width(5.dp).fillMaxHeight().background(color))
            Column(Modifier.padding(horizontal = 12.dp, vertical = 10.dp)) {
                Text(
                    (if (withDate) "${shortDate(s.date)} · " else "") + "${clock(s.time)} · Ibadah ${s.typeName}",
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(s.title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                val people = listOfNotNull(
                    s.leader.ifEmpty { null }?.let { "Khadim: $it" },
                    s.mc.ifEmpty { null }?.let { "Pembawa acara: $it" },
                ).joinToString(" · ")
                if (people.isNotEmpty()) {
                    Text(people, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}

@Composable
fun ColorDot(color: Color, size: Int = 10) {
    Box(Modifier.size(size.dp).clip(CircleShape).background(color))
}

/** Small rounded label, e.g. "Baptis". */
@Composable
fun Pill(text: String, container: Color = MaterialTheme.colorScheme.surfaceVariant) {
    Surface(shape = RoundedCornerShape(50), color = container) {
        Text(text, style = MaterialTheme.typography.labelMedium, modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp))
    }
}
