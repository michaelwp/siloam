package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MemberFilterTest {
    private fun m(baptized: Boolean?, confirmed: Boolean?, status: String = "", gender: String = "") = Member(
        id = "m", familyId = "f", name = "M", familyName = "X", kolom = 11, gender = gender, maritalStatus = status,
        baptized = baptized, confirmed = confirmed, category = "", flags = emptyList(), birthMonth = null, birthDay = null,
    )

    @Test
    fun matchesEachField() {
        assertTrue(MemberFilter.none.matches(m(null, null)))
        assertFalse(MemberFilter.none.active)
        val sidi = MemberFilter(confirmed = Tri.YES)
        assertTrue(sidi.matches(m(true, true)))
        assertFalse(sidi.matches(m(true, false)))
        assertFalse(sidi.matches(m(true, null)))
        assertTrue(MemberFilter(baptized = Tri.UNKNOWN).matches(m(null, true)))
        assertTrue(MemberFilter(baptized = Tri.NO).matches(m(false, null)))
        val married = MemberFilter(baptized = Tri.YES, status = "Menikah")
        assertTrue(married.matches(m(true, true, "Menikah")))
        assertFalse(married.matches(m(true, true, "Belum Menikah")))
        assertFalse(married.matches(m(false, true, "Menikah")))
        val women = MemberFilter(gender = GENDER_FEMALE)
        assertTrue(women.active)
        assertTrue(women.matches(m(null, null, gender = GENDER_FEMALE)))
        assertFalse(women.matches(m(null, null, gender = GENDER_MALE)))
    }

    @Test
    fun survivesSaving() {
        val f = MemberFilter(Tri.YES, Tri.UNKNOWN, "Cerai Hidup / Duda", GENDER_MALE)
        assertEquals(f, MemberFilter.decode(f.encode()))
        assertEquals(MemberFilter(Tri.YES, status = "Menikah"), MemberFilter.decode("yes||Menikah")) // saved before gender
        assertEquals(MemberFilter.none, MemberFilter.decode(MemberFilter.none.encode()))
        assertEquals(MemberFilter.none, MemberFilter.decode("junk"))
    }

    @Test
    fun offersKnownThenOtherStatuses() {
        val s = maritalStatuses(listOf(m(null, null, "Menikah"), m(null, null, "Lainnya"), m(null, null)))
        assertEquals(MARITAL_STATUSES + "Lainnya", s)
    }
}
