package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Test

class StatsTest {
    private fun m(name: String, gender: String, category: String, elderly: Boolean = false, month: Int? = null) = Member(
        id = name, familyId = "f", name = name, familyName = "X", kolom = 11, gender = gender, maritalStatus = "",
        baptized = null, confirmed = null, category = category, flags = if (elderly) listOf(LANSIA) else emptyList(), birthMonth = month, birthDay = month?.let { 1 },
    )

    private fun f(size: Int) = Family(id = "f$size-${size * 7}", kolom = 11, no = 1, name = "F", note = "", memberCount = size, weddingMonth = null, weddingDay = null)

    @Test
    fun countsLikeTheWeb() {
        val members = listOf(
            m("a", GENDER_MALE, "Anak", month = 1),
            m("b", GENDER_FEMALE, "Anak", month = 1),
            m("c", GENDER_FEMALE, "WKI", elderly = true, month = 10),
            m("d", GENDER_MALE, "PKB", elderly = true),
            m("e", GENDER_MALE, ""), // no birth date → no category
        )
        val s = stats(members, listOf(f(1), f(3), f(3)))
        assertEquals(3, s.families)
        assertEquals(5, s.members)
        assertEquals(3, s.male)
        assertEquals(2, s.female)
        assertEquals(2, s.bipra["Anak"])
        assertEquals(1, s.bipra["WKI"])
        assertEquals(2, s.bipra[LANSIA]) // Lansia counts on top of PKB/WKI
        assertEquals(1, s.uncategorized)
        assertEquals(GenderSplit(1, 1), s.bipraByGender.first { it.first.key == "Anak" }.second)
        assertEquals(2, s.birthdaysPerMonth[0])
        assertEquals(1, s.birthdaysPerMonth[9])
        assertEquals(listOf(1, 0, 2), s.familySizes)
    }

    @Test
    fun countsBaptismAndSidi() {
        val members = listOf(
            m("a", GENDER_MALE, "Anak").copy(baptized = true, confirmed = false),
            m("b", GENDER_FEMALE, "Remaja").copy(baptized = true, confirmed = true),
            m("c", GENDER_FEMALE, "WKI").copy(baptized = false),
        )
        val s = stats(members, emptyList())
        assertEquals(YesNo(yes = 2, no = 1, unknown = 0), s.baptized)
        assertEquals(YesNo(yes = 1, no = 1, unknown = 1), s.confirmed)
    }

    @Test
    fun emptyData() {
        val s = stats(emptyList(), emptyList())
        assertEquals(emptyList<Int>(), s.familySizes)
        assertEquals(List(12) { 0 }, s.birthdaysPerMonth)
    }

    @Test
    fun countsTheServersCategories() {
        val cats = listOf(Category("Pemudi", "Pemudi", "17 – 30"), Category("Baru", "Baru menikah", "≤ 35 / menikah", isFlag = true))
        val members = listOf(
            m("a", GENDER_FEMALE, "Pemudi"),
            m("b", GENDER_FEMALE, "Pemudi").copy(flags = listOf("Baru")),
            m("c", GENDER_MALE, "PKB"), // not in this list
        )
        val s = stats(members, emptyList(), cats)
        assertEquals(mapOf("Pemudi" to 2, "Baru" to 1), s.bipra)
        assertEquals(listOf("Pemudi"), s.bipraByGender.map { it.first.key }) // flags are not in the gender chart
    }
}
