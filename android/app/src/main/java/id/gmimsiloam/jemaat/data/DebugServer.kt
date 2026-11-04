package id.gmimsiloam.jemaat.data

import java.net.URI

/** Debug builds: servers to pick from, besides typing one. */
val DEBUG_SERVERS = listOf(
    "http://localhost:8080" to "Komputer ini lewat adb reverse (tcp:8080)",
    "http://10.0.2.2:8080" to "Komputer ini dari emulator",
)

/**
 * The server address typed in the debug setting, cleaned ("http://host:port",
 * no trailing slash), or null when it is not an http(s) address.
 */
fun debugServerUrl(input: String): String? {
    val v = input.trim().trimEnd('/')
    val uri = runCatching { URI(v) }.getOrNull() ?: return null
    if (uri.scheme !in setOf("http", "https") || uri.host.isNullOrEmpty()) return null
    if (!uri.query.isNullOrEmpty() || !uri.fragment.isNullOrEmpty()) return null
    return v
}
