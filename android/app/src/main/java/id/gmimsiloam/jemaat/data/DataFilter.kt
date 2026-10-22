package id.gmimsiloam.jemaat.data

/**
 * What the app shows and notifies about — the "filter data bawaan", chosen in
 * the top bar or in Pengaturan: a kolom (its families and members, its services
 * and the whole congregation's) and/or a BIPRA category (its members, their
 * families and the services of its jenis); both together narrow to the
 * category's members in that kolom. A jenis ibadah ([typeId]) narrows the
 * services further. A kolom, category or jenis that no longer exists (or a
 * jenis not of the chosen category) counts as not chosen.
 */
data class DataFilter(val kolom: Int? = null, val bipra: String = "", val bipraId: String = "", val typeId: String = "") {
    companion object {
        val all = DataFilter()
        fun kolom(k: Int) = DataFilter(kolom = k)

        /** A category, by id (stays right when an admin changes its code) and code (older servers have no ids). */
        fun bipra(c: Category) = DataFilter(bipra = c.key, bipraId = c.id)
    }

    /** The same category, another kolom (null: every kolom). */
    fun withKolom(k: Int?) = copy(kolom = k)

    /** The same kolom, another category (null: every category). */
    fun withBipra(c: Category?) = copy(bipra = c?.key ?: "", bipraId = c?.id ?: "")

    /** The same kolom and category, another jenis (null: every jenis). */
    fun withType(t: WorshipType?) = copy(typeId = t?.id ?: "")

    /** Whether this filter is about category [c]: by id when both have one, else by code. */
    fun isCategory(c: Category) =
        (bipra.isNotEmpty() || bipraId.isNotEmpty()) &&
            if (bipraId.isNotEmpty() && c.id.isNotEmpty()) c.id == bipraId else c.key == bipra

    /** The category this filter is about, if one is chosen and exists. */
    fun category(data: Snapshot): Category? = data.categories.firstOrNull { isCategory(it) }

    /** The kolom this filter is about, if one is chosen and exists. */
    fun kolomOf(data: Snapshot): Int? = data.effectiveKolom(kolom)

    /**
     * The services: of the kolom (and the whole congregation's, kolom 0), and of a
     * category's linked jenis only.
     */
    fun services(data: Snapshot): List<Service> {
        val inKolom = data.services.forKolom(kolomOf(data))
        type(data)?.let { t -> return inKolom.filter { it.typeId == t.id } }
        if (category(data) == null) return inKolom
        val types = typeChoices(data).map { it.id }.toSet()
        return inKolom.filter { it.typeId in types }
    }

    /** The jenis to choose from: those linked to the chosen category, or all. */
    fun typeChoices(data: Snapshot): List<WorshipType> {
        val c = category(data) ?: return data.types
        return data.types.filter { it.bipraId.isNotEmpty() && it.bipraId == c.id }
    }

    /** The jenis this filter is about, if one is chosen and is among [typeChoices]. */
    fun type(data: Snapshot): WorshipType? =
        typeId.takeIf { it.isNotEmpty() }?.let { id -> typeChoices(data).firstOrNull { it.id == id } }

    /** The members (also for birthdays). */
    fun members(data: Snapshot): List<Member> {
        val inKolom = data.members.inKolom(kolomOf(data)) { it.kolom }
        val c = category(data) ?: return inKolom
        return inKolom.filter { inCategory(it, c) }
    }

    /** The families of the kolom; with a category, those with a member in it. */
    fun families(data: Snapshot): List<Family> {
        val inKolom = data.families.inKolom(kolomOf(data)) { it.kolom }
        if (category(data) == null) return inKolom
        val ids = members(data).map { m -> m.familyId }.toSet()
        return inKolom.filter { f -> f.id in ids }
    }

    /** As [label], shorter with both ("K11 · Pemuda"), for the top bar. */
    fun shortLabel(data: Snapshot?): String {
        val k = if (data == null) kolom else kolomOf(data)
        val c = if (data == null) bipra.ifEmpty { null } else category(data)?.label
        return if (k != null && c != null) "K$k · $c" else label(data)
    }

    /** "Semua Kolom", "Kolom 11", "Pemuda" or "Kolom 11 · Pemuda". */
    fun label(data: Snapshot?): String {
        val k = if (data == null) kolom else kolomOf(data)
        val c = if (data == null) bipra.ifEmpty { null } else category(data)?.label
        return when {
            c == null -> k?.let { "Kolom $it" } ?: "Semua Kolom"
            k == null -> c
            else -> "Kolom $k · $c"
        }
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
