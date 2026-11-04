@file:OptIn(ExperimentalMaterial3Api::class)

package id.gmimsiloam.jemaat.ui

import android.Manifest
import android.app.Activity
import android.os.Build
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.OutlinedButton
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
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
import id.gmimsiloam.jemaat.data.Scope
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
    // what the screens show: the Kolom and BIPRA pickers (a kolom or category that no longer
    // exists: everything, see DataFilter); the notifications have their own filter (Pengaturan)
    val filter = state.view

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

    // visitors: the statistics summary and the sign-in form only
    if (!state.signedIn) {
        PublicApp(state, vm)
        return
    }

    if (settings) {
        SettingsScreen(state = state, vm = vm, onBack = { settings = false })
        return
    }

    Scaffold(
        topBar = {
            Column {
            TopAppBar(
                title = {
                    Column {
                        Text("GMIM Siloam", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                        Text(tab.label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                },
                actions = {
                    IconButton(onClick = { settings = true }) { Icon(Icons.Filled.Settings, contentDescription = "Pengaturan") }
                    IconButton(onClick = { confirmClose = true }) {
                        Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = "Tutup aplikasi")
                    }
                },
            )
            Pickers(filter, state.data, onPick = vm::setView)
            }
        },
        bottomBar = {
            NavigationBar {
                // a BIPRA-only user has no family pages: no BIPRA tab (as on the web)
                Tab.entries.filterNot { it == Tab.Bipra && state.data?.scope?.bipraOnly == true }.forEach { t ->
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
                    // a BIPRA-only user has no BIPRA tab: a category tap opens the Jemaat tab
                    val bipraTab = !data.scope.bipraOnly
                    val shown = if (tab == Tab.Bipra && !bipraTab) Tab.Home else tab
                    tabState.SaveableStateProvider(shown.key) {
                        when (shown) {
                            Tab.Home -> HomeScreen(
                                data, filter, state.error, state.update,
                                onCategory = {
                                    if (bipraTab) {
                                        category = it
                                        tab = Tab.Bipra
                                    } else {
                                        membersOpened++
                                        tab = Tab.Members
                                    }
                                },
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
 * What the screens show, picked separately as on the web: a kolom (or all) and
 * a BIPRA category (or all), so both can apply. A limited user's choices follow
 * their role (see [Scope]): their koloms (every kolom with categories), and in
 * one of their own koloms every category, elsewhere only their own.
 */
@Composable
private fun Pickers(filter: DataFilter, data: Snapshot?, onPick: (DataFilter) -> Unit) {
    if (data == null) return
    val scope = data.scope
    val kolom = filter.kolomOf(data)
    val category = filter.category(data)
    val categories = scope.bipraChoices(data, kolom)
    val ownOnly = categories.size < data.categories.size
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp).padding(bottom = 6.dp),
    ) {
        Picker(
            label = kolomLabel(kolom).let { if (kolom == null && scope.limited) "Semua Data Saya" else it },
            fixed = scope.singleKolom != null,
            modifier = Modifier.weight(1f),
        ) { close ->
            if (scope.singleKolom == null) {
                MenuChoice(if (scope.limited) "Semua Data Saya" else "Semua Kolom", kolom == null) { onPick(filter.withKolom(null)); close() }
            }
            data.koloms.forEach { k ->
                MenuChoice(kolomLabel(k.number), kolom == k.number) { onPick(filter.withKolom(k.number)); close() }
            }
        }
        if (categories.isNotEmpty()) {
            val all = if (ownOnly) "Semua BIPRA Saya" else "Semua BIPRA"
            Picker(
                label = category?.label ?: all,
                // a single own category outside their koloms: nothing to pick
                fixed = ownOnly && categories.size == 1,
                modifier = Modifier.weight(1f),
            ) { close ->
                MenuChoice(all, category == null) { onPick(filter.withBipra(null)); close() }
                categories.forEach { c ->
                    MenuChoice(c.label, category == c) { onPick(filter.withBipra(c)); close() }
                }
            }
        }
    }
}

/** A dropdown button showing [label]; [fixed]: just the label, nothing to pick. */
@Composable
private fun Picker(label: String, fixed: Boolean, modifier: Modifier, items: @Composable (close: () -> Unit) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box(modifier) {
        OutlinedButton(onClick = { open = true }, enabled = !fixed, modifier = Modifier.fillMaxWidth()) {
            Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
            if (!fixed) Icon(Icons.Filled.ArrowDropDown, contentDescription = null)
        }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            items { open = false }
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


fun kolomLabel(k: Int?) = if (k == null) "Semua Kolom" else "Kolom $k"
