package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.LocalDate

class DataFilterTest {
    private fun service(id: String, kolom: Int, type: String) = Service(
        id = id, kolom = kolom, typeId = type, typeName = type, typeColor = "#039be5",
        date = LocalDate.of(2026, 10, 22), time = "19:00", host = "", location = "",
        leader = "", mc = "", note = "", start = "", end = "",
    )

    private fun member(name: String, kolom: Int, category: String) = Member(
        id = name, familyId = "f$kolom", name = name, familyName = "X", kolom = kolom, gender = "",
        maritalStatus = "", baptized = null, confirmed = null, category = category, flags = emptyList(),
        birthMonth = 10, birthDay = 22,
    )

    private fun family(id: String, kolom: Int) =
        Family(id = id, kolom = kolom, no = 1, name = id, note = "", memberCount = 1, weddingMonth = null, weddingDay = null)

    private val pemuda = Category("Pemuda", "Pemuda", "17 – 30", id = "cat-pemuda")
    private val data = Snapshot(
        koloms = listOf(Kolom("k3", 3), Kolom("k11", 11)),
        types = listOf(WorshipType("kolom", "Kolom", "#039be5"), WorshipType("pemuda", "Pemuda", "#f4511e", bipraId = "cat-pemuda")),
        families = listOf(family("f3", 3), family("f11", 11), family("f99", 11)),
        members = listOf(member("Budi", 3, "Pemuda"), member("Erwin", 11, "PKB"), member("Sari", 11, "Pemuda")),
        services = listOf(service("k3", 3, "kolom"), service("k11", 11, "kolom"), service("pem", 0, "pemuda"), service("pem3", 3, "pemuda")),
        categories = listOf(pemuda, Category("PKB", "PKB", "31+ / menikah", id = "cat-pkb")),
        fetchedAt = 0,
    )

    @Test
    fun filtersByKolomAndCategory() {
        val all = DataFilter.all
        assertEquals(4, all.services(data).size)
        assertEquals(3, all.members(data).size)
        assertEquals("Semua Kolom", all.label(data))

        // a category: the linked jenis in every kolom, and the category's members
        val bipra = DataFilter.bipra(pemuda)
        assertEquals(listOf("pem", "pem3"), bipra.services(data).map { it.id })
        assertEquals(listOf("Budi", "Sari"), bipra.members(data).map { it.name })
        assertEquals(listOf("f3", "f11"), bipra.families(data).map { it.id }) // the families they are in
        assertEquals("Pemuda", bipra.label(data))

        // a kolom: that kolom's and the whole congregation's services
        val k3 = DataFilter.kolom(3)
        assertEquals(listOf("k3", "pem", "pem3"), k3.services(data).map { it.id })
        assertEquals(listOf("Budi"), k3.members(data).map { it.name })
        assertEquals(listOf("f3"), k3.families(data).map { it.id })
        assertEquals("Kolom 3", k3.label(data))

        // both: the category's members in the kolom; its jenis in the kolom and for the whole congregation
        val both = DataFilter.kolom(11).withBipra(pemuda)
        assertEquals(listOf("Sari"), both.members(data).map { it.name })
        assertEquals(listOf("f11"), both.families(data).map { it.id })
        assertEquals(listOf("pem"), both.services(data).map { it.id })
        assertEquals("Kolom 11 · Pemuda", both.label(data))
        assertEquals("K11 · Pemuda", both.shortLabel(data))
        assertEquals("Pemuda", bipra.shortLabel(data))
        // each part changes on its own
        assertEquals(DataFilter.bipra(pemuda), both.withKolom(null))
        assertEquals(DataFilter.kolom(11), both.withBipra(null))

        // a deleted kolom or category counts as not chosen
        assertEquals(4, DataFilter.kolom(99).services(data).size)
        assertEquals(3, DataFilter(bipra = "Hilang", bipraId = "cat-gone").members(data).size)
        assertEquals("Pemuda", DataFilter(kolom = 99, bipra = "Pemuda", bipraId = "cat-pemuda").label(data))

        // an admin renamed the code: the id still finds it
        val renamed = data.copy(categories = listOf(pemuda.copy(key = "PMD", label = "Pemuda Gereja")))
        assertEquals("Pemuda Gereja", bipra.label(renamed))
        // a filter saved before ids (or an older server): by code
        assertEquals("Pemuda", DataFilter(bipra = "Pemuda").label(data))
    }

    @Test
    fun jenisChoicesFollowTheCategory() {
        // the Jadwal tab's jenis: all, or the chosen category's linked ones
        assertEquals(data.types, DataFilter.kolom(3).typeChoices(data))
        assertEquals(listOf(data.types[1]), DataFilter.bipra(pemuda).typeChoices(data))
        assertEquals(emptyList<WorshipType>(), DataFilter.bipra(Category("PKB", "PKB", "", id = "cat-pkb")).typeChoices(data))
    }

    @Test
    fun groupsPerKolomCongregationFirst() {
        val groups = data.services.byKolom { it.kolom }
        assertEquals(listOf(0, 3, 11), groups.map { it.first })
        assertEquals(listOf("k3", "pem3"), groups[1].second.map { it.id })
        assertEquals("Semua kolom", kolomHeading(0))
    }
}
