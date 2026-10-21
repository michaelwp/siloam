package id.gmimsiloam.jemaat.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance

// The web app's colors (accent, surfaces, green for birthdays).
private val Light = lightColorScheme(
    primary = Color(0xFF1F5FAE),
    onPrimary = Color.White,
    primaryContainer = Color(0xFFE6EEFA),
    onPrimaryContainer = Color(0xFF0F2F57),
    secondary = Color(0xFF17683D),
    secondaryContainer = Color(0xFFE3F4EA),
    onSecondaryContainer = Color(0xFF0B3520),
    background = Color(0xFFF5F6F8),
    surface = Color.White,
    surfaceVariant = Color(0xFFF0F2F5),
    onSurfaceVariant = Color(0xFF667085),
    outlineVariant = Color(0xFFE2E5EA),
)

private val Dark = darkColorScheme(
    primary = Color(0xFF6EA3E8),
    onPrimary = Color(0xFF0B1E36),
    primaryContainer = Color(0xFF1D2B40),
    onPrimaryContainer = Color(0xFFD6E4F8),
    secondary = Color(0xFF6FCF97),
    secondaryContainer = Color(0xFF173225),
    onSecondaryContainer = Color(0xFFCDEEDB),
    background = Color(0xFF12151B),
    surface = Color(0xFF1A1E26),
    surfaceVariant = Color(0xFF222733),
    onSurfaceVariant = Color(0xFF98A2B3),
    outlineVariant = Color(0xFF2E3442),
)

@Composable
fun SiloamTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = if (isSystemInDarkTheme()) Dark else Light, content = content)
}

/**
 * Text color on a jenis color: black or white, whichever contrasts more (the
 * web's palette has light colors such as Kuning, where white is unreadable).
 */
fun onTypeColor(background: Color): Color {
    val l = background.luminance()
    return if ((l + 0.05f) / 0.05f >= 1.05f / (l + 0.05f)) Color(0xDE000000) else Color.White
}

/** A worship type's calendar color ("#039be5"), or the theme's primary. */
@Composable
fun typeColor(hex: String): Color =
    runCatching { Color(android.graphics.Color.parseColor(hex)) }.getOrElse { MaterialTheme.colorScheme.primary }
