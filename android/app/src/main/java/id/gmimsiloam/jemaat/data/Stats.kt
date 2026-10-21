package id.gmimsiloam.jemaat.data

/** A BIPRA category as the server computes it (the web's CATEGORIES), plus Lansia. */
data class Category(val key: String, val label: String, val range: String)

const val GENDER_MALE = "Laki Laki"
const val GENDER_FEMALE = "Perempuan"
const val LANSIA = "Lansia"

/** Anak (ASM) 0–11, Remaja 12–16, Pemuda 17–30, PKB/WKI 31+ or married. */
val CATEGORIES = listOf(
    Category("Anak", "Anak (ASM)", "0 – 11"),
    Category("Remaja", "Remaja", "12 – 16"),
    Category("Pemuda", "Pemuda", "17 – 30"),
    Category("PKB", "PKB", "31+ / menikah"),
    Category("WKI", "WKI", "31+ / menikah"),
)

/** Lansia (60+) is a flag on top of the category, not a category of its own. */
val LANSIA_CATEGORY = Category(LANSIA, "Lansia", "60+")

/** Whether [m] belongs to the category [key] (or is elderly, for [LANSIA]). */
fun inCategory(m: Member, key: String) = if (key == LANSIA) m.elderly else m.category == key

data class GenderSplit(val male: Int, val female: Int) {
    val total get() = male + female
}

data class Stats(
    val families: Int,
    val members: Int,
    val male: Int,
    val female: Int,
    /** per BIPRA category key, and [LANSIA] */
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

fun stats(members: List<Member>, families: List<Family>): Stats {
    val maxSize = families.maxOfOrNull { it.memberCount } ?: 0
    return Stats(
        families = families.size,
        members = members.size,
        male = members.count { it.gender == GENDER_MALE },
        female = members.count { it.gender == GENDER_FEMALE },
        bipra = (CATEGORIES + LANSIA_CATEGORY).associate { c -> c.key to members.count { inCategory(it, c.key) } },
        uncategorized = members.count { it.category.isEmpty() },
        bipraByGender = CATEGORIES.map { c ->
            val inCat = members.filter { it.category == c.key }
            c to GenderSplit(inCat.count { it.gender == GENDER_MALE }, inCat.count { it.gender == GENDER_FEMALE })
        },
        birthdaysPerMonth = (1..12).map { month -> members.count { it.hasBirthday() && it.birthMonth == month } },
        familySizes = (1..maxSize).map { size -> families.count { it.memberCount == size } },
    )
}
