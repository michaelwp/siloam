package id.gmimsiloam.jemaat.data

import org.json.JSONObject

/** The newest app the server knows of (GET /api/app/android). */
data class AppRelease(val latestVersion: String, val downloadUrl: String)

/**
 * Reads the server's answer; null when no release is configured or the link
 * is not https (the app only ever opens https download links).
 */
fun parseRelease(json: String): AppRelease? {
    val o = JSONObject(json)
    val version = o.optString("latestVersion").trim()
    val url = o.optString("downloadUrl").trim()
    if (version.isEmpty() || !url.startsWith("https://")) return null
    return AppRelease(version, url)
}

/**
 * Whether version [latest] ("1.0.10") is newer than [current] ("1.0.9"),
 * comparing the numbers part by part; missing parts count as 0 ("1.1" =
 * "1.1.0"), and anything after a number ("1.0.0-debug") is ignored.
 */
fun isNewer(latest: String, current: String): Boolean {
    fun parts(v: String) = v.trim().split('.').map { p -> p.takeWhile { it.isDigit() }.toIntOrNull() ?: 0 }
    val a = parts(latest)
    val b = parts(current)
    for (i in 0 until maxOf(a.size, b.size)) {
        val x = a.getOrElse(i) { 0 }
        val y = b.getOrElse(i) { 0 }
        if (x != y) return x > y
    }
    return false
}
