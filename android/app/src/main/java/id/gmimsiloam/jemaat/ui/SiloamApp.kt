@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import android.Manifest
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import id.gmimsiloam.jemaat.AppViewModel
import id.gmimsiloam.jemaat.notify.DailyWorker

enum class Tab(val key: String, val label: String, val icon: ImageVector) {
    Home("home", "Beranda", Icons.Filled.Home),
    Schedule("schedule", "Jadwal", Icons.Filled.DateRange),
    Birthdays("birthdays", "Ulang Tahun", Icons.Filled.Star),
    Members("members", "Jemaat", Icons.Filled.Person),
}

@Composable
fun SiloamApp(requestedTab: String?, onTabHandled: () -> Unit, vm: AppViewModel = viewModel()) {
    val state by vm.state.collectAsStateWithLifecycle()
    var tab by rememberSaveable { mutableStateOf(Tab.Home) }
    var settings by rememberSaveable { mutableStateOf(false) }

    // a tapped notification opens its tab
    LaunchedEffect(requestedTab) {
        Tab.entries.firstOrNull { it.key == requestedTab }?.let {
            tab = it
            settings = false
        }
        if (requestedTab != null) onTabHandled()
    }

    // Android 13+: ask once for permission to show notifications
    val askNotifications = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) DailyWorker.runNow(vm.getApplication())
    }
    LaunchedEffect(Unit) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            askNotifications.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }

    if (settings) {
        SettingsScreen(state = state, vm = vm, onBack = { settings = false })
        return
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("GMIM Siloam", style = MaterialTheme.typography.titleLarge)
                        Text(tab.label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                },
                actions = {
                    KolomMenu(kolom = state.kolom, koloms = state.data?.koloms?.map { it.number } ?: emptyList(), onPick = vm::setKolom)
                    IconButton(onClick = { settings = true }) { Icon(Icons.Filled.Settings, contentDescription = "Pengaturan") }
                },
            )
        },
        bottomBar = {
            NavigationBar {
                Tab.entries.forEach { t ->
                    NavigationBarItem(
                        selected = tab == t,
                        onClick = { tab = t },
                        icon = { Icon(t.icon, contentDescription = null) },
                        label = { Text(t.label) },
                    )
                }
            }
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = state.refreshing,
            onRefresh = vm::refresh,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            val data = state.data
            when {
                data == null && state.refreshing -> Message("Memuat data…")
                data == null -> Message(
                    "Data belum dapat dimuat.\n${state.error ?: ""}\n\nPeriksa koneksi internet, lalu tarik ke bawah untuk mencoba lagi.",
                )
                else -> Box(Modifier.fillMaxSize()) {
                    when (tab) {
                        Tab.Home -> HomeScreen(data, state.kolom, state.error)
                        Tab.Schedule -> ScheduleScreen(data, state.kolom)
                        Tab.Birthdays -> BirthdaysScreen(data, state.kolom)
                        Tab.Members -> MembersScreen(data, state.kolom)
                    }
                }
            }
        }
    }
}

/** "Semua Kolom" / "Kolom 11" picker; the choice is also used for notifications. */
@Composable
private fun KolomMenu(kolom: Int?, koloms: List<Int>, onPick: (Int?) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box {
        TextButton(onClick = { open = true }) {
            Text(kolomLabel(kolom))
            Icon(Icons.Filled.ArrowDropDown, contentDescription = null)
        }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            DropdownMenuItem(text = { Text(kolomLabel(null)) }, onClick = { onPick(null); open = false })
            koloms.forEach { k ->
                DropdownMenuItem(text = { Text(kolomLabel(k)) }, onClick = { onPick(k); open = false })
            }
        }
    }
}

fun kolomLabel(k: Int?) = if (k == null) "Semua Kolom" else "Kolom $k"
