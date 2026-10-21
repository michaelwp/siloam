package id.gmimsiloam.jemaat

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import id.gmimsiloam.jemaat.data.AppRelease
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
    /** viewed kolom; null = all */
    val kolom: Int? = null,
    /** a newer app version to offer, if the last check found one */
    val update: AppRelease? = null,
)

class AppViewModel(app: Application) : AndroidViewModel(app) {
    private val prefs = Prefs(app)
    private val repo = Repository(app)
    private val _state = MutableStateFlow(UiState(kolom = prefs.kolom, update = prefs.availableUpdate(BuildConfig.VERSION_NAME)))
    val state: StateFlow<UiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            repo.cached()?.let { cached -> _state.update { it.copy(data = cached) } }
            refresh()
        }
    }

    private var refreshJob: Job? = null

    /** Loads fresh data; [restart] drops a running load (e.g. it is from the old server). */
    fun refresh(restart: Boolean = false) {
        if (_state.value.refreshing && !restart) return
        refreshJob?.cancel()
        _state.update { it.copy(refreshing = true) }
        refreshJob = viewModelScope.launch {
            val result = runCatching { repo.refresh() }
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
                    onSuccess = { s.copy(data = it, refreshing = false, error = null, update = prefs.availableUpdate(BuildConfig.VERSION_NAME)) },
                    onFailure = { e ->
                        s.copy(refreshing = false, error = e.message ?: "Gagal memuat data", update = prefs.availableUpdate(BuildConfig.VERSION_NAME))
                    },
                )
            }
        }
    }

    /** The kolom both the screens and the notifications follow. */
    fun setKolom(k: Int?) {
        prefs.kolom = k
        _state.update { it.copy(kolom = k) }
        // reminders queued for the old kolom's services go; the new kolom's come
        DailyWorker.rescheduleReminders(getApplication())
    }

    fun setServerUrl(url: String) {
        prefs.serverUrl = url
        refresh(restart = true)
    }
}
