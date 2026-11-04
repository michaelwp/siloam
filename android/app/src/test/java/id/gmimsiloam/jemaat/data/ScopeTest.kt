package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.LocalDate

class ScopeTest {
    private val pemuda = Category("Pemuda", "Pemuda", "17 – 30", id = "p")
    private val pkb = Category("PKB", "PKB", "31+", id = "k")

    private fun m(id: String, kolom: Int, category: String) = Member(
        id = id, familyId = "f$kolom", name = id, familyName = "X", kolom = kolom, gender = GENDER_MALE, maritalStatus = "",
        baptized = null, confirmed = null, category = category, flags = emptyList(), birthMonth = null, birthDay = null,
    )

    private fun f(kolom: Int) = Family("f$kolom", kolom, 1, "F$kolom", "", 1, null, null)

    private fun s(id: String, kolom: Int, type: String) =
        Service(id, kolom, type, type, "", LocalDate.of(2026, 11, 11), "19:00", "", "", "", "", "", "", "")

    private val data = Snapshot(
        koloms = listOf(Kolom("1", 11), Kolom("2", 4)),
        types = listOf(WorshipType("kolom", "Kolom", ""), WorshipType("pemuda", "Pemuda", "", "p"), WorshipType("pkb", "PKB", "", "k")),
        families = listOf(f(11), f(4)),
        members = listOf(m("ani", 11, "PKB"), m("budi", 4, "Pemuda"), m("erwin", 4, "PKB")),
        services = listOf(s("k11", 11, "kolom"), s("k4", 4, "kolom"), s("p4", 4, "pemuda"), s("pkb4", 4, "pkb"), s("p0", 0, "pemuda")),
        categories = listOf(pemuda, pkb),
        fetchedAt = 0,
    )

    @Test
    fun koloms11PlusPemudaEverywhere() {
        val d = Scope(listOf(11), listOf("p")).apply(data)
        assertEquals(listOf("ani", "budi"), d.members.map { it.id })
        assertEquals(listOf("f11", "f4"), d.families.map { it.id })
        assertEquals(listOf("k11", "p4", "p0"), d.services.map { it.id })
        assertEquals(2, d.koloms.size) // their category's members are in every kolom
        // in Kolom 11 any category, elsewhere only theirs
        assertEquals(listOf(pemuda, pkb), d.scope.bipraChoices(d, 11))
        assertEquals(listOf(pemuda), d.scope.bipraChoices(d, 4))
        assertEquals(listOf(pemuda), d.scope.bipraChoices(d, null))
        assertNull(DataFilter(kolom = 4, bipra = "PKB", bipraId = "k").category(d)) // PKB of Kolom 4 is not theirs
    }

    @Test
    fun singleKolomIsFixed() {
        val d = Scope(listOf(11)).apply(data)
        assertEquals(listOf(11), d.koloms.map { it.number })
        assertEquals(11, DataFilter.all.kolomOf(d))
        assertEquals(listOf("k11", "p0"), d.services.map { it.id }) // and the whole congregation's, as on the web
    }

    @Test
    fun everythingForUnlimited() {
        assertEquals(data, Scope.none.apply(data))
        assertEquals(Scope(listOf(3, 5), listOf("a")), Scope.decode(Scope(listOf(3, 5), listOf("a")).encode()))
        assertEquals(Scope.none, Scope.decode(""))
    }
}
