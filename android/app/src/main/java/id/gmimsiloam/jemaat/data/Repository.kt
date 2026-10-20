package id.gmimsiloam.jemaat.data

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.File

/**
 * Fetches the data and keeps the last good copy on the device, so the app
 * opens (and notifications work) without a connection.
 */
class Repository(private val context: Context) {
    private val cacheFile = File(context.filesDir, "snapshot.json")

    /** The cached snapshot, or null when nothing was fetched yet (or it is unreadable). */
    suspend fun cached(): Snapshot? = withContext(Dispatchers.IO) {
        runCatching {
            if (!cacheFile.exists()) return@runCatching null
            val o = JSONObject(cacheFile.readText())
            RawData(
                koloms = o.getString("koloms"),
                types = o.getString("types"),
                families = o.getString("families"),
                members = o.getString("members"),
                services = o.getString("services"),
            ).parse(o.getLong("fetchedAt"))
        }.getOrNull()
    }

    /** Fetches fresh data from the server and caches it; throws on failure. */
    suspend fun refresh(): Snapshot = withContext(Dispatchers.IO) {
        val raw = Api(Prefs(context).serverUrl).fetchAll()
        val now = System.currentTimeMillis()
        val snapshot = raw.parse(now) // parse before caching: never cache a bad body
        val o = JSONObject()
            .put("koloms", raw.koloms)
            .put("types", raw.types)
            .put("families", raw.families)
            .put("members", raw.members)
            .put("services", raw.services)
            .put("fetchedAt", now)
        val tmp = File(context.filesDir, "snapshot.json.tmp")
        tmp.writeText(o.toString())
        tmp.renameTo(cacheFile)
        snapshot
    }

    /** Fresh data when the server answers, else the cached copy (null if none). */
    suspend fun refreshOrCached(): Snapshot? = runCatching { refresh() }.getOrElse { cached() }
}
