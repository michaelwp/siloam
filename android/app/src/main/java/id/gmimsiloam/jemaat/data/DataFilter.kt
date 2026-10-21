package id.gmimsiloam.jemaat.data

/**
 * What the app shows and notifies about — the "filter data bawaan", chosen in
 * the top bar or in Pengaturan: everything, one BIPRA category (its members,
 * their families and the services of its jenis, in every kolom), or one kolom
 * (its families and members, its services and the whole congregation's). A
 * kolom or category that no longer exists falls back to everything.
 */
data class DataFilter(val mode: String, val kolom: Int?, val bipra: String, val bipraId: String = "") {
    companion object {
        const val ALL = "all"
        const val BIPRA = "bipra"
        const val KOLOM = "kolom"

        val all = DataFilter(ALL, null, "")
        fun kolom(k: Int) = DataFilter(KOLOM, k, "")

        /** A category, by id (stays right when an admin changes its code) and code (older servers have no ids). */
        fun bipra(c: Category) = DataFilter(BIPRA, null, c.key, c.id)
    }

    /** Whether this filter is about category [c]: by id when both have one, else by code. */
    fun isCategory(c: Category) =
        mode == BIPRA && if (bipraId.isNotEmpty() && c.id.isNotEmpty()) c.id == bipraId else c.key == bipra

    /** The category this filter is about, if it is a BIPRA filter whose category exists. */
    fun category(data: Snapshot): Category? = data.categories.firstOrNull { isCategory(it) }

    /** The kolom this filter is about, if it is a kolom filter whose kolom exists. */
    fun kolomOf(data: Snapshot): Int? =
        if (mode == KOLOM) data.effectiveKolom(kolom) else null

    /**
     * The services. Per BIPRA: those whose jenis is linked to the category (in
     * every kolom). Per kolom: that kolom's and the whole congregation's (kolom 0).
     */
    fun services(data: Snapshot): List<Service> {
        category(data)?.let { c ->
            val types = data.types.filter { it.bipraId.isNotEmpty() && it.bipraId == c.id }.map { it.id }.toSet()
            return data.services.filter { it.typeId in types }
        }
        return data.services.forKolom(kolomOf(data))
    }

    /** The members (also for birthdays). */
    fun members(data: Snapshot): List<Member> {
        category(data)?.let { c -> return data.members.filter { inCategory(it, c) } }
        return data.members.inKolom(kolomOf(data)) { it.kolom }
    }

    /** The families: per BIPRA, those with a member in the category. */
    fun families(data: Snapshot): List<Family> {
        category(data)?.let {
            val ids = members(data).map { m -> m.familyId }.toSet()
            return data.families.filter { f -> f.id in ids }
        }
        return data.families.inKolom(kolomOf(data)) { it.kolom }
    }

    /** "Semua Kolom", "Pemuda" or "Kolom 11". */
    fun label(data: Snapshot?): String {
        if (data == null) {
            return when (mode) {
                BIPRA -> bipra
                KOLOM -> kolom?.let { "Kolom $it" } ?: "Semua Kolom"
                else -> "Semua Kolom"
            }
        }
        category(data)?.let { return it.label }
        return kolomOf(data)?.let { "Kolom $it" } ?: "Semua Kolom"
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
