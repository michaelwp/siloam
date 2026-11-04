package id.gmimsiloam.jemaat

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import id.gmimsiloam.jemaat.data.AppRelease
import id.gmimsiloam.jemaat.data.DataFilter
import id.gmimsiloam.jemaat.data.LoginResult
import id.gmimsiloam.jemaat.data.NotSignedInException
import id.gmimsiloam.jemaat.data.PublicStats
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Repository
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.notify.DailyWorker
import kotlinx.coroutines.Job
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class UiState(
    val data: Snapshot? = null,
    val refreshing: Boolean = false,
    /** the last refresh failed (the cached data, if any, is still shown) */
    val error: String? = null,
    /** what the notifications are about (Pengaturan; default: all of the user's data) */
    val filter: DataFilter = DataFilter.all,
    /** what the screens show (the Kolom and BIPRA pickers in the top bar) */
    val view: DataFilter = DataFilter.all,
    /** a newer app version to offer, if the last check found one */
    val update: AppRelease? = null,
    /** signed in: the jemaat data; else (a visitor) only [publicStats] */
    val signedIn: Boolean = false,
    /** who is signed in ("budi · Kelola Kolom 11") */
    val account: String = "",
    /** the statistics summary visitors see */
    val publicStats: PublicStats? = null,
    val loggingIn: Boolean = false,
    val loginError: String? = null,
    /** signed in with a temporary password: the username to change it for (see changeCredentials) */
    val mustChange: String? = null,
)

class AppViewModel(app: Application) : AndroidViewModel(app) {
    private val prefs = Prefs(app)
    private val repo = Repository(app)
    private val _state = MutableStateFlow(
        UiState(
            filter = prefs.filter,
            view = prefs.viewFilter,
            update = prefs.availableUpdate(BuildConfig.VERSION_NAME),
            signedIn = prefs.signedIn,
            account = prefs.account,
        ),
    )
    val state: StateFlow<UiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            // signed in to a server address typed in an older version: sign in again here
            if (repo.dropOldServer()) {
                _state.update { it.copy(signedIn = false, account = "", error = "Alamat server aplikasi diperbarui. Silakan masuk lagi.") }
            }
            repo.cached()?.let { cached -> _state.update { it.copy(data = cached) } }
            refresh()
        }
    }

    private var refreshJob: Job? = null

    /** Loads fresh data; [restart] drops a running load (e.g. one started before signing in or out). */
    fun refresh(restart: Boolean = false) {
        if (_state.value.refreshing && !restart) return
        refreshJob?.cancel()
        _state.update { it.copy(refreshing = true) }
        if (!prefs.signedIn) {
            // a visitor: the statistics summary only
            refreshJob = viewModelScope.launch {
                val stats = repo.publicStats()
                repo.checkRelease()
                ensureActive()
                _state.update {
                    it.copy(
                        publicStats = stats ?: it.publicStats,
                        refreshing = false,
                        error = if (stats == null) "Statistik belum dapat dimuat" else null,
                        update = prefs.availableUpdate(BuildConfig.VERSION_NAME),
                    )
                }
            }
            return
        }
        refreshJob = viewModelScope.launch {
            val result = runCatching { repo.refresh() }
            // the session ended (expired, or an admin signed this device out): back to the visitor view
            if (result.exceptionOrNull() is NotSignedInException) {
                signedOut(result.exceptionOrNull()?.message)
                return@launch
            }
            // replaced by a newer load (runCatching also caught the cancellation):
            // leave the state to that one, never show "Job was cancelled"
            ensureActive()
            // the update check rides along when the server answered (it never fails the load)
            if (result.isSuccess) {
                repo.checkRelease()
                ensureActive()
            }
            _state.update { s ->
                result.fold(
                    // the account line follows a role change made by an admin
                    onSuccess = {
                        s.copy(data = it, refreshing = false, error = null, account = prefs.account, update = prefs.availableUpdate(BuildConfig.VERSION_NAME))
                    },
                    onFailure = { e ->
                        s.copy(refreshing = false, error = e.message ?: "Gagal memuat data", update = prefs.availableUpdate(BuildConfig.VERSION_NAME))
                    },
                )
            }
        }
    }

    /** What the screens show (the top bar's pickers); the notifications keep their own filter. */
    fun setView(f: DataFilter) {
        prefs.viewFilter = f
        _state.update { it.copy(view = f) }
    }

    /** What the notifications are about (Pengaturan). */
    fun setFilter(f: DataFilter) {
        prefs.filter = f
        _state.update { it.copy(filter = f) }
        // today's reminders were queued for the old filter's services
        DailyWorker.rescheduleReminders(getApplication())
    }

    fun login(username: String, password: String) {
        if (_state.value.loggingIn) return
        _state.update { it.copy(loggingIn = true, loginError = null) }
        viewModelScope.launch {
            val result = runCatching { repo.login(username.trim(), password) }
            result.fold(
                onSuccess = { r ->
                    when (r) {
                        is LoginResult.SignedIn -> signedIn(r.account)
                        is LoginResult.MustChange -> {
                            pendingSession = r.session
                            _state.update { it.copy(loggingIn = false, mustChange = r.username) }
                        }
                    }
                },
                onFailure = { e -> _state.update { it.copy(loggingIn = false, loginError = e.message ?: "Gagal masuk") } },
            )
        }
    }

    /** The session of a sign-in that must change its temporary password first (not kept on the device). */
    private var pendingSession: String? = null

    /** Sets the own username and password (first sign-in, or after a reset), as on the web. */
    fun changeCredentials(currentPassword: String, username: String, newPassword: String) {
        val session = pendingSession ?: return
        if (_state.value.loggingIn) return
        _state.update { it.copy(loggingIn = true, loginError = null) }
        viewModelScope.launch {
            runCatching { repo.changeCredentials(session, currentPassword, username.trim(), newPassword) }.fold(
                onSuccess = { account ->
                    pendingSession = null
                    signedIn(account)
                },
                onFailure = { e ->
                    // the session ended meanwhile (e.g. locked after wrong passwords): sign in again
                    if (e is NotSignedInException) {
                        pendingSession = null
                        _state.update { it.copy(loggingIn = false, mustChange = null, loginError = "Sesi berakhir, silakan masuk lagi") }
                    } else {
                        _state.update { it.copy(loggingIn = false, loginError = e.message ?: "Gagal menyimpan") }
                    }
                },
            )
        }
    }

    /** Leaves the password change: back to the sign-in form (the session is ended). */
    fun cancelChange() {
        val session = pendingSession
        pendingSession = null
        _state.update { it.copy(mustChange = null, loginError = null) }
        if (session != null) viewModelScope.launch { repo.abandon(session) }
    }

    private fun signedIn(account: String) {
        _state.update {
            it.copy(
                loggingIn = false, signedIn = true, account = account, error = null, mustChange = null, loginError = null,
                filter = prefs.filter, view = prefs.viewFilter,
            )
        }
        refresh(restart = true)
        DailyWorker.rescheduleReminders(getApplication())
    }

    /** Debug builds: another server ([url]; "" = the built-in one); signs out and loads from there. */
    fun setDebugServer(url: String) {
        if (!BuildConfig.DEBUG) return
        refreshJob?.cancel()
        viewModelScope.launch {
            repo.switchServer(url)
            _state.update { it.copy(publicStats = null, mustChange = null, loginError = null) }
            pendingSession = null
            signedOut(null)
        }
    }

    fun logout() {
        viewModelScope.launch {
            repo.logout()
            signedOut(null)
        }
    }

    /** Back to the visitor view: no data, no notifications (queued reminders dropped). */
    private fun signedOut(message: String?) {
        refreshJob?.cancel()
        _state.update { it.copy(signedIn = false, account = "", data = null, refreshing = false, error = message) }
        DailyWorker.rescheduleReminders(getApplication())
        refresh(restart = true)
    }

}
