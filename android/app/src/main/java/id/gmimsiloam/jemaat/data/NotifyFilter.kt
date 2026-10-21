package id.gmimsiloam.jemaat.data

/**
 * What the notifications (daily summary and reminders) are about, chosen in
 * Pengaturan: everything, one BIPRA category, or one kolom. A kolom or
 * category that no longer exists falls back to everything.
 */
data class NotifyFilter(val mode: String, val kolom: Int?, val bipra: String) {
    companion object {
        const val ALL = "all"
        const val BIPRA = "bipra"
        const val KOLOM = "kolom"
    }

    /** The category this filter is about, if it is a BIPRA filter whose category exists. */
    fun category(data: Snapshot): Category? =
        if (mode == BIPRA) data.categories.firstOrNull { it.key == bipra } else null

    private fun kolomOf(data: Snapshot): Int? =
        if (mode == KOLOM) data.effectiveKolom(kolom) else null

    /**
     * The services to notify about. Per BIPRA: those whose jenis is linked to
     * the category (in every kolom). Per kolom: that kolom's and the whole
     * congregation's (kolom 0).
     */
    fun services(data: Snapshot): List<Service> {
        category(data)?.let { c ->
            val types = data.types.filter { it.bipraId.isNotEmpty() && it.bipraId == c.id }.map { it.id }.toSet()
            return data.services.filter { it.typeId in types }
        }
        return data.services.forKolom(kolomOf(data))
    }

    /** The members whose birthdays to notify about. */
    fun members(data: Snapshot): List<Member> {
        category(data)?.let { c -> return data.members.filter { inCategory(it, c) } }
        return data.members.inKolom(kolomOf(data)) { it.kolom }
    }
}

/** "Semua kolom" (the whole congregation) or "Kolom 11". */
fun kolomHeading(kolom: Int) = if (kolom == 0) "Semua kolom" else "Kolom $kolom"

/**
 * Items per kolom, "Semua kolom" (0) first, then Kolom 1, 2, …; a single
 * group when all items are of one kolom (then there is no need for headings).
 */
fun <T> List<T>.byKolom(kolomOf: (T) -> Int): List<Pair<Int, List<T>>> =
    groupBy(kolomOf).toSortedMap().map { (k, v) -> k to v }
