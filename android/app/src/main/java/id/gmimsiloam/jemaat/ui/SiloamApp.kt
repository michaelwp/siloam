@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import android.Manifest
import android.app.Activity
import android.os.Build
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.material.icons.filled.Check
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.saveable.rememberSaveableStateHolder
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import id.gmimsiloam.jemaat.AppViewModel
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.MemberFilter
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.notify.DailyWorker
import id.gmimsiloam.jemaat.notify.Notifier

enum class Tab(val key: String, val label: String, val icon: ImageVector) {
    Home("home", "Beranda", Icons.Filled.Home),
    Schedule("schedule", "Jadwal", Icons.Filled.DateRange),
    Birthdays("birthdays", "HUT", Icons.Filled.Cake),
    Members("members", "Jemaat", Icons.Filled.Person),
    Bipra("bipra", "BIPRA", Icons.AutoMirrored.Filled.List),
}

@Composable
fun SiloamApp(requestedTab: String?, onTabHandled: () -> Unit, vm: AppViewModel = viewModel()) {
    val state by vm.state.collectAsStateWithLifecycle()
    var tab by rememberSaveable { mutableStateOf(Tab.Home) }
    var settings by rememberSaveable { mutableStateOf(false) }
    var confirmClose by rememberSaveable { mutableStateOf(false) }
    var category by rememberSaveable { mutableStateOf<String?>(null) } // BIPRA tab's filter
    var members by rememberSaveable { mutableStateOf(MemberFilter.none.encode()) } // Jemaat tab's filters
    var membersOpened by rememberSaveable { mutableIntStateOf(0) } // a tile opened the Jemaat tab: a fresh search

    // a tapped notification opens its tab
    LaunchedEffect(requestedTab) {
        Tab.entries.firstOrNull { it.key == requestedTab }?.let {
            tab = it
            settings = false
        }
        if (requestedTab != null) onTabHandled()
    }

    // Android 13+: ask once for permission to show notifications; after
    // that, Pengaturan links to the system settings
    val askNotifications = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) DailyWorker.runNow(vm.getApplication())
    }
    val context = LocalContext.current
    LaunchedEffect(Unit) {
        val prefs = Prefs(context)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU && !prefs.askedNotificationPermission && !Notifier.canPost(context)) {
            prefs.askedNotificationPermission = true
            askNotifications.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }
    // the "filter data bawaan" (a kolom or category that no longer exists: everything, see DataFilter)
    val filter = state.filter

    // Back from another tab returns to Beranda before leaving the app
    BackHandler(enabled = !settings && tab != Tab.Home) { tab = Tab.Home }
    // each tab keeps its saved state (month, search, filters) while another is shown
    val tabState = rememberSaveableStateHolder()

    if (confirmClose) {
        AlertDialog(
            onDismissRequest = { confirmClose = false },
            title = { Text("Tutup aplikasi?") },
            text = { Text("Notifikasi ibadah dan ulang tahun tetap muncul setelah aplikasi ditutup.") },
            confirmButton = {
                TextButton(onClick = {
                    confirmClose = false
                    // closes the screens and removes the app from the recent apps; the
                    // background check (WorkManager) keeps sending notifications
                    (context as? Activity)?.finishAndRemoveTask()
                }) { Text("Tutup") }
            },
            dismissButton = { TextButton(onClick = { confirmClose = false }) { Text("Batal") } },
        )
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
                    FilterMenu(filter, state.data, onPick = vm::setFilter)
                    IconButton(onClick = { settings = true }) { Icon(Icons.Filled.Settings, contentDescription = "Pengaturan") }
                    IconButton(onClick = { confirmClose = true }) {
                        Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = "Tutup aplikasi")
                    }
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
                    tabState.SaveableStateProvider(tab.key) {
                        when (tab) {
                            Tab.Home -> HomeScreen(
                                data, filter, state.error, state.update,
                                onCategory = { category = it; tab = Tab.Bipra },
                                onMembers = {
                                    members = it.encode()
                                    membersOpened++
                                    tab = Tab.Members
                                },
                            )
                            Tab.Schedule -> ScheduleScreen(data, filter)
                            Tab.Birthdays -> BirthdaysScreen(data, filter)
                            Tab.Members -> MembersScreen(data, filter, MemberFilter.decode(members), membersOpened, onMemberFilter = { members = it.encode() })
                            Tab.Bipra -> BipraScreen(data, filter, category, onCategory = { category = it })
                        }
                    }
                }
            }
        }
    }
}

/**
 * The "filter data bawaan": a kolom (or all), a BIPRA category (or all) and a jenis
 * ibadah (or all), each picked on its own, so they combine. The screens and the
 * notifications follow it; it is also in Pengaturan.
 */
@Composable
private fun FilterMenu(filter: DataFilter, data: Snapshot?, onPick: (DataFilter) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box {
        TextButton(onClick = { open = true }, modifier = Modifier.widthIn(max = 180.dp)) {
            // a long category name must not push the title away
            Text(filter.shortLabel(data), maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f, fill = false))
            Icon(Icons.Filled.ArrowDropDown, contentDescription = null)
        }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            // a pick changes its own part only; the ticks show both
            val kolom = data?.let { filter.kolomOf(it) }
            val category = data?.let { filter.category(it) }
            // the few categories first; the koloms (often a dozen) after
            data?.categories?.takeIf { it.isNotEmpty() }?.let { categories ->
                MenuHeading("Kategori BIPRA")
                MenuChoice("Semua BIPRA", category == null) { onPick(filter.withBipra(null)); open = false }
                categories.forEach { c ->
                    MenuChoice(c.label, category == c) { onPick(filter.withBipra(c)); open = false }
                }
                HorizontalDivider()
            }
            // the jenis (of the chosen category's, when one is chosen): narrows the services only
            data?.let { d -> filter.typeChoices(d) }?.takeIf { it.isNotEmpty() }?.let { types ->
                val type = filter.type(data)
                MenuHeading("Jenis Ibadah")
                MenuChoice("Semua Jenis Ibadah", type == null) { onPick(filter.withType(null)); open = false }
                types.forEach { t ->
                    MenuChoice(t.name, type == t) { onPick(filter.withType(t)); open = false }
                }
                HorizontalDivider()
            }
            MenuHeading("Kolom")
            MenuChoice("Semua Kolom", kolom == null) { onPick(filter.withKolom(null)); open = false }
            data?.koloms?.forEach { k ->
                MenuChoice(kolomLabel(k.number), kolom == k.number) { onPick(filter.withKolom(k.number)); open = false }
            }
        }
    }
}

/** A menu item with a tick when [selected]. */
@Composable
private fun MenuChoice(text: String, selected: Boolean, onClick: () -> Unit) {
    DropdownMenuItem(
        text = { Text(text, fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal) },
        leadingIcon = {
            if (selected) Icon(Icons.Filled.Check, contentDescription = "dipilih") else Spacer(Modifier.size(24.dp))
        },
        onClick = onClick,
    )
}

@Composable
private fun MenuHeading(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.labelMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
    )
}

fun kolomLabel(k: Int?) = if (k == null) "Semua Kolom" else "Kolom $k"
