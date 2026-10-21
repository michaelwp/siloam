package id.gmimsiloam.jemaat.data

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.File
import java.io.IOException

/**
 * Fetches the data and keeps the last good copy on the device, so the app
 * opens (and notifications work) without a connection.
 */
class Repository(private val context: Context) {
    private companion object {
        /** One writer of the cache file at a time (screens and background work share the process). */
        val cacheLock = Mutex()
    }


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
                categories = if (o.has("categories")) o.getString("categories") else null,
            ).parse(o.getLong("fetchedAt"))
        }.getOrNull()
    }

    /**
     * Fetches fresh data from the server and caches it; throws on failure.
     * The screens and the background check may load at the same time, and a
     * cancelled load still finishes its (blocking) download: the cache is
     * written under one lock, and only while [Prefs.serverUrl] is still the
     * server the data came from.
     */
    suspend fun refresh(): Snapshot = withContext(Dispatchers.IO) {
        val server = Prefs(context).serverUrl
        val raw = Api(server).fetchAll()
        val now = System.currentTimeMillis()
        val snapshot = raw.parse(now) // parse before caching: never cache a bad body
        val o = JSONObject()
            .put("koloms", raw.koloms)
            .put("types", raw.types)
            .put("families", raw.families)
            .put("members", raw.members)
            .put("services", raw.services)
            .apply { raw.categories?.let { put("categories", it) } }
            .put("fetchedAt", now)
        cacheLock.withLock {
            if (Prefs(context).serverUrl != server) throw IOException("Alamat server berubah saat memuat data")
            val tmp = File.createTempFile("snapshot", ".tmp", context.filesDir)
            try {
                tmp.writeText(o.toString())
                if (!tmp.renameTo(cacheFile)) throw IOException("Data tidak dapat disimpan")
            } finally {
                tmp.delete() // no-op after a successful rename
            }
        }
        snapshot
    }

    /**
     * Asks the server for the newest app release and remembers it. A failed
     * check keeps what the last one found; a server without a configured
     * release clears it.
     */
    suspend fun checkRelease() = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        val api = Api(prefs.serverUrl)
        val body = runCatching { api.get("/api/app/android") }.getOrNull() ?: return@withContext
        val release = runCatching { parseRelease(body) }.getOrNull()
        prefs.latestVersion = release?.latestVersion ?: ""
        prefs.downloadUrl = release?.downloadUrl ?: ""
    }

    /** Fresh data when the server answers, else the cached copy (null if none). */
    suspend fun refreshOrCached(): Snapshot? = runCatching { refresh() }.getOrElse { cached() }
}
