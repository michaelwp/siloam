package id.gmimsiloam.jemaat.data

import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

/** Reads the public, read-only endpoints of the Siloam server. */
class Api(private val baseUrl: String) {

    /** GET [path] and return the body; throws IOException on any failure. */
    fun get(path: String): String {
        val conn = URL(baseUrl.trimEnd('/') + path).openConnection() as HttpURLConnection
        try {
            conn.connectTimeout = 15_000
            conn.readTimeout = 20_000
            conn.setRequestProperty("Accept", "application/json")
            conn.instanceFollowRedirects = true
            val code = conn.responseCode
            if (code !in 200..299) throw IOException("Server menjawab $code untuk $path")
            return conn.inputStream.bufferedReader(Charsets.UTF_8).use { it.readText() }
        } finally {
            conn.disconnect()
        }
    }

    /** Fetches everything the app shows, as the raw JSON bodies. */
    fun fetchAll(): RawData = RawData(
        koloms = get("/api/koloms"),
        types = get("/api/worship-types"),
        families = get("/api/families"),
        members = get("/api/members"),
        services = get("/api/services"),
        // older servers have fixed categories (no endpoint): the defaults
        categories = runCatching { get("/api/bipra-categories") }.getOrNull(),
    )
}

/** The JSON bodies of one fetch (also what the offline cache stores). */
data class RawData(
    val koloms: String,
    val types: String,
    val families: String,
    val members: String,
    val services: String,
    /** null: the server has no editable categories (DEFAULT_CATEGORIES apply) */
    val categories: String?,
) {
    fun parse(fetchedAt: Long) = Snapshot(
        koloms = Parse.koloms(koloms),
        types = Parse.types(types),
        families = Parse.families(families),
        members = Parse.members(members),
        services = Parse.services(services),
        categories = categories?.let(Parse::categories) ?: DEFAULT_CATEGORIES,
        fetchedAt = fetchedAt,
    )
}
