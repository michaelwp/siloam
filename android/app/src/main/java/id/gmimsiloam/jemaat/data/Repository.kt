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
    private val statsFile = File(context.filesDir, "stats.json")

    private fun api(prefs: Prefs = Prefs(context)) = Api(prefs.serverUrl, prefs.session.ifEmpty { null })

    /**
     * Signs in with a username and password; returns the account line shown in
     * Pengaturan. An account that must first change its password is told to
     * do that on the web (and is not kept signed in).
     */
    suspend fun login(username: String, password: String): String = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        val res = Api(prefs.serverUrl).post("/api/auth/login", JSONObject().put("username", username).put("password", password))
        val token = res.sessionCookie ?: throw IOException("Server tidak memberi sesi masuk")
        val user = JSONObject(res.body)
        if (user.optBoolean("mustChangePassword")) {
            runCatching { Api(prefs.serverUrl, token).post("/api/auth/logout") }
            throw IOException("Akun ini wajib mengganti username/password terlebih dahulu. Masuk lewat web untuk menggantinya, lalu coba lagi.")
        }
        val account = user.optString("username") + " · " + roleLabel(user)
        prefs.session = token
        prefs.account = account
        account
    }

    /** Signs out here (and on the server), and forgets the data on the device. */
    suspend fun logout() = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        if (prefs.signedIn) runCatching { api(prefs).post("/api/auth/logout") }
        forget(prefs)
    }

    /** Drops the session and the cached data (signed out, or the session ended). */
    private suspend fun forget(prefs: Prefs) {
        prefs.session = ""
        prefs.account = ""
        cacheLock.withLock { cacheFile.delete() }
    }

    /** The statistics summary visitors see (cached for offline use); null if never fetched. */
    suspend fun publicStats(): PublicStats? = withContext(Dispatchers.IO) {
        val body = runCatching { Api(Prefs(context).serverUrl).get("/api/stats") }.getOrNull()
        if (body != null) {
            val parsed = runCatching { parsePublicStats(body) }.getOrNull()
            if (parsed != null) {
                runCatching { statsFile.writeText(body) }
                return@withContext parsed
            }
        }
        runCatching { if (statsFile.exists()) parsePublicStats(statsFile.readText()) else null }.getOrNull()
    }

    /** The cached snapshot, or null when nothing was fetched yet (or it is unreadable). */
    suspend fun cached(): Snapshot? = withContext(Dispatchers.IO) {
        runCatching {
            // jemaat data only for a signed-in user
            if (!Prefs(context).signedIn || !cacheFile.exists()) return@runCatching null
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
        val prefs = Prefs(context)
        if (!prefs.signedIn) throw NotSignedInException("Belum masuk")
        val server = prefs.serverUrl
        val raw = try {
            api(prefs).fetchAll()
        } catch (e: NotSignedInException) {
            forget(prefs) // the session ended (expired, or an admin signed this device out)
            throw e
        }
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
            if (!Prefs(context).signedIn) throw NotSignedInException("Sudah keluar")
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

/** "Admin", "Pengguna", "Kelola Kolom 3, 5 · BIPRA (2)", "Hanya lihat", as on the web (category names are not known yet). */
private fun roleLabel(user: JSONObject): String {
    val koloms = user.optJSONArray("koloms")?.let { a -> (0 until a.length()).map { a.optInt(it) } } ?: emptyList()
    val bipras = user.optJSONArray("bipraIds")?.length() ?: 0
    val parts = listOfNotNull(
        koloms.takeIf { it.isNotEmpty() }?.let { "Kolom " + it.joinToString(", ") },
        bipras.takeIf { it > 0 }?.let { "BIPRA ($it)" },
    )
    return when (user.optString("role")) {
        "admin" -> "Admin"
        "user" -> "Pengguna"
        "viewer" -> if (parts.isEmpty()) "Hanya lihat" else "Lihat " + parts.joinToString(" · ")
        else -> "Kelola " + parts.joinToString(" · ")
    }
}
