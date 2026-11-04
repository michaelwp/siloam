package id.gmimsiloam.jemaat.data

/**
 * What a signed-in user may see, as on the web: admins and regular users (and
 * a viewer with no choices) everything; a user limited to some koloms and/or
 * BIPRA categories ("kelola kolom/BIPRA", or a viewer with choices) their
 * koloms' families, members and services (and the whole congregation's
 * services), plus their categories' members (and the families they are in)
 * and services in every kolom. The server redacts the rest; the app does not
 * show it at all.
 */
data class Scope(val koloms: List<Int> = emptyList(), val bipraIds: List<String> = emptyList()) {
    companion object {
        val none = Scope()

        /** From the signed-in user's JSON (role, koloms, bipraIds). */
        fun of(user: org.json.JSONObject): Scope {
            if (user.optString("role") !in setOf("scoped", "viewer", "treasurer")) return none
            val ks = user.optJSONArray("koloms")?.let { a -> (0 until a.length()).map { a.optInt(it) } } ?: emptyList()
            val bs = user.optJSONArray("bipraIds")?.let { a -> (0 until a.length()).map { a.optString(it) } } ?: emptyList()
            return Scope(ks.filter { it > 0 }.sorted(), bs.filter { it.isNotEmpty() })
        }

        /** As stored in [Prefs] ("3,5|id1,id2"). */
        fun decode(s: String): Scope {
            if (s.isEmpty()) return none
            val (k, b) = (s.split('|') + "").take(2)
            return Scope(k.split(',').mapNotNull { it.toIntOrNull() }, b.split(',').filter { it.isNotEmpty() })
        }
    }

    fun encode() = if (!limited) "" else koloms.joinToString(",") + "|" + bipraIds.joinToString(",")

    /** limited to some koloms and/or categories */
    val limited get() = koloms.isNotEmpty() || bipraIds.isNotEmpty()

    /** only categories, no kolom: no family pages (no BIPRA tab, no wedding anniversaries) */
    val bipraOnly get() = koloms.isEmpty() && bipraIds.isNotEmpty()

    /** a single kolom and no categories: no "all" choice in the kolom filter */
    val singleKolom get() = koloms.singleOrNull()?.takeIf { bipraIds.isEmpty() }

    /** The user's own categories, in order. */
    fun ownCategories(data: Snapshot) = data.categories.filter { it.id in bipraIds }

    /**
     * The BIPRA filter's choices for the viewed [kolom]: every category in one of
     * their own koloms (or for an unlimited user), elsewhere (and over all of
     * their data) only their own categories.
     */
    fun bipraChoices(data: Snapshot, kolom: Int?): List<Category> =
        if (bipraIds.isEmpty() || (kolom != null && kolom in koloms)) data.categories else ownCategories(data)

    /** [data] narrowed to what this user may see (unchanged for an unlimited user). */
    fun apply(data: Snapshot): Snapshot {
        if (!limited) return data
        val cats = ownCategories(data)
        val members = data.members.filter { m -> m.kolom in koloms || cats.any { inCategory(m, it) } }
        val familyIds = members.map { it.familyId }.toSet()
        val linked = data.types.filter { it.bipraId.isNotEmpty() && it.bipraId in bipraIds }.map { it.id }.toSet()
        return data.copy(
            // with categories, their members are in every kolom
            koloms = if (bipraIds.isEmpty()) data.koloms.filter { it.number in koloms } else data.koloms,
            families = data.families.filter { it.kolom in koloms || it.id in familyIds },
            members = members,
            services = data.services.filter { s ->
                s.kolom in koloms || (koloms.isNotEmpty() && s.kolom == 0) || s.typeId in linked
            },
            scope = this,
        )
    }
}
