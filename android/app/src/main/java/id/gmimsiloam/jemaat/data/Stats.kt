package id.gmimsiloam.jemaat.data

/** A Kategorial BIPRA category (managed by admins on the web); [isFlag]: an extra mark like Lansia. */
data class Category(val key: String, val label: String, val range: String, val isFlag: Boolean = false, val id: String = "")

const val GENDER_MALE = "Laki Laki"
const val GENDER_FEMALE = "Perempuan"
const val LANSIA = "Lansia"

/** The categories of servers without editable ones: Anak (ASM) 0–11, …, PKB/WKI 31+ or married, Lansia 60+. */
val DEFAULT_CATEGORIES = listOf(
    Category("Anak", "Anak (ASM)", "0 – 11"),
    Category("Remaja", "Remaja", "12 – 16"),
    Category("Pemuda", "Pemuda", "17 – 30"),
    Category("PKB", "PKB", "31+ / menikah"),
    Category("WKI", "WKI", "31+ / menikah"),
    Category(LANSIA, "Lansia", "60+", isFlag = true),
)

/** The short age hint of a rule, as on the web: "0 – 11", "31+ / menikah", "60+". */
fun bipraRange(minAge: Int?, maxAge: Int?, married: Boolean): String {
    val age = when {
        minAge != null && maxAge != null -> "$minAge – $maxAge"
        minAge != null -> "$minAge+"
        maxAge != null -> "≤ $maxAge"
        else -> ""
    }
    return if (married) (if (age.isEmpty()) "menikah" else "$age / menikah") else age
}

/** Whether [m] is in [c]: its main category, or one of its flags. */
fun inCategory(m: Member, c: Category) = if (c.isFlag) c.key in m.flags else m.category == c.key

data class GenderSplit(val male: Int, val female: Int) {
    val total get() = male + female
}

/** Baptized or confirmed (sidi): [yes], [no] (not yet), [unknown] (no data). */
data class YesNo(val yes: Int, val no: Int, val unknown: Int)

private fun yesNo(values: List<Boolean?>) = YesNo(values.count { it == true }, values.count { it == false }, values.count { it == null })

data class Stats(
    val families: Int,
    val members: Int,
    val male: Int,
    val female: Int,
    /** per marital status (the known ones, then any other the data has) */
    val byStatus: List<Pair<String, Int>>,
    val baptized: YesNo,
    /** sidi */
    val confirmed: YesNo,
    /** per BIPRA category key (main categories and flags) */
    val bipra: Map<String, Int>,
    /** members without a birth date have no category */
    val uncategorized: Int,
    /** per category: men and women */
    val bipraByGender: List<Pair<Category, GenderSplit>>,
    /** index 0 = January */
    val birthdaysPerMonth: List<Int>,
    /** index 0 = families of 1 member, up to the largest family */
    val familySizes: List<Int>,
)

fun stats(members: List<Member>, families: List<Family>, categories: List<Category> = DEFAULT_CATEGORIES): Stats {
    val maxSize = families.maxOfOrNull { it.memberCount } ?: 0
    return Stats(
        families = families.size,
        members = members.size,
        male = members.count { it.gender == GENDER_MALE },
        female = members.count { it.gender == GENDER_FEMALE },
        byStatus = maritalStatuses(members).map { st -> st to members.count { it.maritalStatus == st } },
        baptized = yesNo(members.map { it.baptized }),
        confirmed = yesNo(members.map { it.confirmed }),
        bipra = categories.associate { c -> c.key to members.count { inCategory(it, c) } },
        uncategorized = members.count { it.category.isEmpty() },
        bipraByGender = categories.filterNot { it.isFlag }.map { c ->
            val inCat = members.filter { it.category == c.key }
            c to GenderSplit(inCat.count { it.gender == GENDER_MALE }, inCat.count { it.gender == GENDER_FEMALE })
        },
        birthdaysPerMonth = (1..12).map { month -> members.count { it.hasBirthday() && it.birthMonth == month } },
        familySizes = (1..maxSize).map { size -> families.count { it.memberCount == size } },
    )
}

/** The statistics summary visitors get (GET /api/stats), with its categories (in order). */
data class PublicStats(val stats: Stats, val categories: List<Category>)

/** Parses /api/stats: the whole congregation's counts, no names. */
fun parsePublicStats(json: String): PublicStats {
    val o = org.json.JSONObject(json)
    fun yn(key: String) = o.getJSONObject(key).let { YesNo(it.optInt("yes"), it.optInt("no"), it.optInt("unknown")) }
    fun ints(key: String) = o.optJSONArray(key)?.let { a -> (0 until a.length()).map { a.optInt(it) } } ?: emptyList()
    val catArray = o.optJSONArray("categories") ?: org.json.JSONArray()
    val categories = Parse.categories(catArray.toString())
    val counts = (0 until catArray.length()).map { catArray.getJSONObject(it) }
    val statuses = o.optJSONArray("maritalStatuses") ?: org.json.JSONArray()
    return PublicStats(
        stats = Stats(
            families = o.optInt("families"),
            members = o.optInt("members"),
            male = o.optInt("male"),
            female = o.optInt("female"),
            byStatus = (0 until statuses.length()).map { statuses.getJSONObject(it) }.map { it.optString("label") to it.optInt("value") },
            baptized = yn("baptized"),
            confirmed = yn("confirmed"),
            bipra = categories.zip(counts).associate { (c, j) -> c.key to j.optInt("count") },
            uncategorized = 0,
            bipraByGender = categories.zip(counts).filterNot { it.first.isFlag }
                .map { (c, j) -> c to GenderSplit(j.optInt("male"), j.optInt("female")) },
            birthdaysPerMonth = ints("birthMonths").let { if (it.size == 12) it else List(12) { 0 } },
            familySizes = ints("familySizes"),
        ),
        categories = categories,
    )
}
