package id.gmimsiloam.jemaat.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.requiredSize
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import id.gmimsiloam.jemaat.AppViewModel
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.R
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.withTimeoutOrNull

/** The splash stays at least this long, so the version can be read. */
private const val SPLASH_MIN_MS = 1_200L

/**
 * …and at most this long: a first load without a connection (or a slow name
 * lookup) can take much longer; the app's own "Memuat data…" / error screen,
 * with pull-to-refresh, is the better place to wait.
 */
private const val SPLASH_MAX_MS = 3_000L

/** The launcher icon's blue (ic_launcher_background). */
private val Brand = Color(0xFF1F5FAE)

/**
 * The app's own splash, laid over the app at a cold start: Android's splash
 * (API 31+) can only show the icon, so this one takes over with the icon at
 * the same size and place, plus the app name and version. It goes after
 * [SPLASH_MIN_MS] once there is something to show (cached data, or the first
 * load has finished), and after [SPLASH_MAX_MS] in any case; [skip] (opened
 * from a notification) leaves it out.
 */
@Composable
fun SplashOverlay(skip: Boolean, vm: AppViewModel = viewModel()) {
    val state by vm.state.collectAsStateWithLifecycle()
    var visible by rememberSaveable { mutableStateOf(!skip) }
    LaunchedEffect(Unit) {
        if (!visible) return@LaunchedEffect
        delay(SPLASH_MIN_MS)
        withTimeoutOrNull(SPLASH_MAX_MS - SPLASH_MIN_MS) {
            snapshotFlow { state.data != null || !state.refreshing }.first { it }
        }
        visible = false
    }
    AnimatedVisibility(visible = visible, exit = fadeOut()) {
        BoxWithConstraints(
            Modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.background)
                // the app underneath must not get the taps
                .pointerInput(Unit) {
                    awaitPointerEventScope {
                        while (true) awaitPointerEvent().changes.forEach { it.consume() }
                    }
                },
        ) {
            // as Android draws an adaptive icon on its splash: the 108dp
            // canvas at 240dp, masked to a 160dp circle, centered on the screen
            Box(Modifier.align(Alignment.Center).size(160.dp).clip(CircleShape).background(Brand), contentAlignment = Alignment.Center) {
                Image(painterResource(R.drawable.ic_launcher_foreground), contentDescription = null, modifier = Modifier.requiredSize(240.dp))
            }
            Column(
                Modifier.align(Alignment.TopCenter).padding(top = maxHeight / 2 + 104.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(stringResource(R.string.app_name), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                Text("GMIM Siloam", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(
                "Versi ${BuildConfig.VERSION_NAME}",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(bottom = 32.dp),
            )
        }
    }
}
