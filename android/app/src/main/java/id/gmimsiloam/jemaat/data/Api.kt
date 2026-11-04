package id.gmimsiloam.jemaat.data

import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

/** The session is gone (expired, signed out elsewhere, or reset by an admin): sign in again. */
class NotSignedInException(message: String = "Sesi berakhir, silakan masuk lagi") : IOException(message)

/** A request refused with the server's message (e.g. a wrong password). */
class ServerMessageException(val status: Int, message: String, val reason: String = "") : IOException(message)

/**
 * Talks to the Siloam server. [session] is the sign-in cookie's value (null:
 * a visitor, who gets only the statistics summary). Every request says it
 * comes from the Android app (X-Client), for the admins' sign-in statistics.
 */
class Api(private val baseUrl: String, private val session: String? = null) {

    /** A response: its body, and the new session cookie the server set (on sign-in). */
    class Response(val body: String, val sessionCookie: String?)

    private fun call(method: String, path: String, body: String? = null): Response {
        val conn = URL(baseUrl.trimEnd('/') + path).openConnection() as HttpURLConnection
        try {
            conn.requestMethod = method
            conn.connectTimeout = 15_000
            conn.readTimeout = 20_000
            conn.setRequestProperty("Accept", "application/json")
            conn.setRequestProperty("X-Client", "android")
            session?.let { conn.setRequestProperty("Cookie", "$SESSION_COOKIE=$it") }
            conn.instanceFollowRedirects = true
            if (body != null) {
                conn.doOutput = true
                conn.setRequestProperty("Content-Type", "application/json")
                conn.outputStream.use { it.write(body.toByteArray(Charsets.UTF_8)) }
            }
            val code = conn.responseCode
            if (code !in 200..299) {
                val text = conn.errorStream?.bufferedReader(Charsets.UTF_8)?.use { it.readText() } ?: ""
                val o = runCatching { JSONObject(text) }.getOrNull()
                val msg = o?.optString("error")?.takeIf { it.isNotEmpty() } ?: "Server menjawab $code untuk $path"
                val reason = o?.optString("code") ?: ""
                if (code == 401) throw NotSignedInException()
                throw ServerMessageException(code, msg, reason)
            }
            val cookie = conn.headerFields.entries
                .filter { it.key.equals("Set-Cookie", ignoreCase = true) }
                .flatMap { it.value }
                .firstNotNullOfOrNull { c ->
                    c.split(';').first().trim().takeIf { it.startsWith("$SESSION_COOKIE=") }?.substringAfter('=')
                }
            val text = if (code == 204) "" else conn.inputStream.bufferedReader(Charsets.UTF_8).use { it.readText() }
            return Response(text, cookie)
        } finally {
            conn.disconnect()
        }
    }

    /** GET [path] and return the body; throws IOException on any failure. */
    fun get(path: String): String = call("GET", path).body

    fun post(path: String, json: JSONObject? = null): Response = call("POST", path, json?.toString() ?: "{}")

    /** Fetches everything the app shows (signed in), as the raw JSON bodies. */
    fun fetchAll(): RawData = RawData(
        koloms = get("/api/koloms"),
        types = get("/api/worship-types"),
        families = get("/api/families"),
        members = get("/api/members"),
        services = get("/api/services"),
        // older servers have fixed categories (no endpoint): the defaults
        categories = runCatching { get("/api/bipra-categories") }.getOrNull(),
        // the kolom cash totals the user may read, with this month's (the phone's); none on older servers
        kas = runCatching { get("/api/kas/summary?month=" + java.time.YearMonth.now()) }.getOrNull(),
        // the BIPRA categories' books likewise; an older server answers with the koloms' (dropped: no bipraId)
        kasBipra = runCatching { get("/api/kas/summary?book=bipra&month=" + java.time.YearMonth.now()) }.getOrNull(),
    )

    companion object {
        const val SESSION_COOKIE = "siloam_session"
    }
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
    /** the Kas Kolom summary; null: none (an older server, or it failed) */
    val kas: String? = null,
    /** the Kas BIPRA summary; null: none */
    val kasBipra: String? = null,
) {
    fun parse(fetchedAt: Long) = Snapshot(
        koloms = Parse.koloms(koloms),
        types = Parse.types(types),
        families = Parse.families(families),
        members = Parse.members(members),
        services = Parse.services(services),
        categories = categories?.let(Parse::categories) ?: DEFAULT_CATEGORIES,
        // never fails the load: the kas card is left out instead
        kas = kas?.let { runCatching { parseKas(it) }.getOrNull() } ?: emptyList(),
        kasBipra = kasBipra?.let { runCatching { parseKas(it).filter { s -> s.bipraId.isNotEmpty() } }.getOrNull() } ?: emptyList(),
        fetchedAt = fetchedAt,
    )
}
