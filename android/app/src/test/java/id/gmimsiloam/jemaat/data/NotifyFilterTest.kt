package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.LocalDate

class NotifyFilterTest {
    private fun service(id: String, kolom: Int, type: String) = Service(
        id = id, kolom = kolom, typeId = type, typeName = type, typeColor = "#039be5",
        date = LocalDate.of(2026, 10, 22), time = "19:00", host = "", location = "",
        leader = "", mc = "", note = "", start = "", end = "",
    )

    private fun member(name: String, kolom: Int, category: String) = Member(
        id = name, familyId = "f", name = name, familyName = "X", kolom = kolom, gender = "",
        maritalStatus = "", baptized = null, confirmed = null, category = category, flags = emptyList(),
        birthMonth = 10, birthDay = 22,
    )

    private val pemuda = Category("Pemuda", "Pemuda", "17 – 30", id = "cat-pemuda")
    private val data = Snapshot(
        koloms = listOf(Kolom("k3", 3), Kolom("k11", 11)),
        types = listOf(WorshipType("kolom", "Kolom", "#039be5"), WorshipType("pemuda", "Pemuda", "#f4511e", bipraId = "cat-pemuda")),
        families = emptyList(),
        members = listOf(member("Budi", 3, "Pemuda"), member("Erwin", 11, "PKB"), member("Sari", 11, "Pemuda")),
        services = listOf(service("k3", 3, "kolom"), service("k11", 11, "kolom"), service("pem", 0, "pemuda"), service("pem3", 3, "pemuda")),
        categories = listOf(pemuda, Category("PKB", "PKB", "31+ / menikah", id = "cat-pkb")),
        fetchedAt = 0,
    )

    @Test
    fun filtersByModes() {
        val all = NotifyFilter(NotifyFilter.ALL, null, "")
        assertEquals(4, all.services(data).size)
        assertEquals(3, all.members(data).size)

        // per BIPRA: the linked jenis in every kolom, and the category's members
        val bipra = NotifyFilter(NotifyFilter.BIPRA, null, "Pemuda")
        assertEquals(listOf("pem", "pem3"), bipra.services(data).map { it.id })
        assertEquals(listOf("Budi", "Sari"), bipra.members(data).map { it.name })

        // per kolom: that kolom's and the whole congregation's services
        val k3 = NotifyFilter(NotifyFilter.KOLOM, 3, "")
        assertEquals(listOf("k3", "pem", "pem3"), k3.services(data).map { it.id })
        assertEquals(listOf("Budi"), k3.members(data).map { it.name })

        // a deleted kolom or category: everything
        assertEquals(4, NotifyFilter(NotifyFilter.KOLOM, 99, "").services(data).size)
        assertEquals(3, NotifyFilter(NotifyFilter.BIPRA, null, "Hilang").members(data).size)
    }

    @Test
    fun groupsPerKolomCongregationFirst() {
        val groups = data.services.byKolom { it.kolom }
        assertEquals(listOf(0, 3, 11), groups.map { it.first })
        assertEquals(listOf("k3", "pem3"), groups[1].second.map { it.id })
        assertEquals("Semua kolom", kolomHeading(0))
    }
}
