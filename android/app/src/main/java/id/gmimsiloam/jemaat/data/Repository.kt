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
     * Signs in with a username and password. A new or reset account must first
     * set its own username and password ([LoginResult.MustChange], as on the
     * web): until then it is not kept signed in on the device.
     */
    suspend fun login(username: String, password: String): LoginResult = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        val res = Api(prefs.serverUrl).post("/api/auth/login", JSONObject().put("username", username).put("password", password))
        val token = res.sessionCookie ?: throw IOException("Server tidak memberi sesi masuk")
        val user = JSONObject(res.body)
        if (user.optBoolean("mustChangePassword")) return@withContext LoginResult.MustChange(token, user.optString("username"))
        LoginResult.SignedIn(keep(prefs, token, user))
    }

    /**
     * Sets the new username and password of an account that must change its
     * temporary one ([session]: from [login]); then it is signed in.
     */
    suspend fun changeCredentials(session: String, currentPassword: String, username: String, newPassword: String): String =
        withContext(Dispatchers.IO) {
            val prefs = Prefs(context)
            val body = JSONObject().put("currentPassword", currentPassword).put("username", username).put("newPassword", newPassword)
            val res = Api(prefs.serverUrl, session).post("/api/auth/change-credentials", body)
            keep(prefs, res.sessionCookie ?: session, JSONObject(res.body))
        }

    /** Signs out a session that never got kept (the password change was cancelled). */
    suspend fun abandon(session: String) = withContext(Dispatchers.IO) {
        runCatching { Api(Prefs(context).serverUrl, session).post("/api/auth/logout") }
    }

    private fun keep(prefs: Prefs, token: String, user: JSONObject): String {
        val account = user.optString("username") + " · " + roleLabel(user)
        prefs.scope = Scope.of(user)
        // a new sign-in starts on all of the user's data, for the screens and the notifications
        prefs.filter = DataFilter.all
        prefs.viewFilter = DataFilter.all
        prefs.session = token
        prefs.account = account
        return account
    }

    /**
     * After updating from a version where another server address was saved: the
     * session (and data) from that server are no good here, so they are
     * forgotten. Returns whether that happened (the user must sign in again).
     */
    suspend fun dropOldServer(): Boolean = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        if (!prefs.dropSavedServer() || !prefs.signedIn) return@withContext false
        forget(prefs)
        true
    }

    /**
     * Debug builds: talks to [url] from now on ("" = the built-in server). The
     * session and the data were the old server's: signed out (best effort on
     * the old server too) and forgotten, also the cached statistics.
     */
    suspend fun switchServer(url: String) = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        if (prefs.signedIn) runCatching { api(prefs).post("/api/auth/logout") }
        forget(prefs)
        statsFile.delete()
        prefs.debugServer = url
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
        prefs.scope = Scope.none
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
                kas = if (o.has("kas")) o.getString("kas") else null,
            ).parse(o.getLong("fetchedAt")).let(Prefs(context).scope::apply)
        }.getOrNull()
    }

    /**
     * Fetches fresh data from the server and caches it; throws on failure.
     * The screens and the background check may load at the same time, and a
     * cancelled load still finishes its (blocking) download: the cache is
     * written under one lock, and only while still signed in.
     */
    suspend fun refresh(): Snapshot = withContext(Dispatchers.IO) {
        val prefs = Prefs(context)
        if (!prefs.signedIn) throw NotSignedInException("Belum masuk")
        val raw = try {
            // the user's role and scope first: an admin may have changed them
            val me = JSONObject(api(prefs).get("/api/auth/me"))
            prefs.scope = Scope.of(me)
            prefs.account = me.optString("username") + " · " + roleLabel(me)
            api(prefs).fetchAll()
        } catch (e: NotSignedInException) {
            forget(prefs) // the session ended (expired, or an admin signed this device out)
            throw e
        }
        val now = System.currentTimeMillis()
        val snapshot = raw.parse(now).let(prefs.scope::apply) // parse before caching: never cache a bad body
        val o = JSONObject()
            .put("koloms", raw.koloms)
            .put("types", raw.types)
            .put("families", raw.families)
            .put("members", raw.members)
            .put("services", raw.services)
            .apply { raw.categories?.let { put("categories", it) } }
            .apply { raw.kas?.let { put("kas", it) } }
            .put("fetchedAt", now)
        cacheLock.withLock {
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

/** "Admin", "Pengguna", "Kelola Kolom 3, 5 · BIPRA (2)", "Hanya lihat", "Bendahara Kolom 3", as on the web (category names are not known yet). */
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
        "treasurer" -> ("Bendahara " + parts.joinToString(" · ")).trim()
        else -> "Kelola " + parts.joinToString(" · ")
    }
}

/** What a sign-in led to. */
sealed interface LoginResult {
    /** signed in; [account]: "budi · Kelola Kolom 11" */
    data class SignedIn(val account: String) : LoginResult

    /** the temporary password must be changed first (session [session], not kept yet) */
    data class MustChange(val session: String, val username: String) : LoginResult
}
