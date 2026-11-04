package id.gmimsiloam.jemaat.data

import org.json.JSONArray

/**
 * A cash book's totals as the server sums them (for the books the user may
 * read): a kolom's (Kas Kolom), or a BIPRA category's (Kas BIPRA: [bipraId]
 * and [name] set, [kolom] 0). The balance, and the cash in and out of [month].
 */
data class KasSummary(
    val kolom: Int,
    val totalIn: Long,
    val totalOut: Long,
    /** "2026-11": the month of [monthIn] and [monthOut] */
    val month: String,
    val monthIn: Long,
    val monthOut: Long,
    val bipraId: String = "",
    val name: String = "",
) {
    val balance: Long get() = totalIn - totalOut
}

fun parseKas(json: String): List<KasSummary> {
    val a = JSONArray(json)
    return (0 until a.length()).map {
        val o = a.getJSONObject(it)
        KasSummary(
            kolom = o.optInt("kolom"),
            totalIn = o.optLong("in"),
            totalOut = o.optLong("out"),
            month = o.optString("month"),
            monthIn = o.optLong("monthIn"),
            monthOut = o.optLong("monthOut"),
            bipraId = o.optString("bipraId"),
            name = o.optString("name"),
        )
    }
}

/** "Rp 1.250.000" ("-Rp 300.000" below zero), as on the web. */
fun rupiah(n: Long): String {
    val digits = kotlin.math.abs(n).toString().reversed().chunked(3).joinToString(".").reversed()
    return (if (n < 0) "-" else "") + "Rp " + digits
}

/** "November 2026" from "2026-11" ("" if it is not a month). */
fun monthLabel(month: String): String {
    val m = month.drop(5).toIntOrNull() ?: return ""
    return if (m in 1..12) "${MONTHS[m - 1]} ${month.take(4)}" else ""
}
