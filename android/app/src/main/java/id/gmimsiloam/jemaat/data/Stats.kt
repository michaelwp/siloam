package id.gmimsiloam.jemaat.data

/** A Kategorial BIPRA category (managed by admins on the web); [isFlag]: an extra mark like Lansia. */
data class Category(val key: String, val label: String, val range: String, val isFlag: Boolean = false)

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

data class Stats(
    val families: Int,
    val members: Int,
    val male: Int,
    val female: Int,
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
