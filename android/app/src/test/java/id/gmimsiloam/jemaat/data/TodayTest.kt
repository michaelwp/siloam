package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.LocalDate

class TodayTest {
    private fun service(id: String, date: String, time: String = "19:00", kolom: Int = 11) = Service(
        id = id, kolom = kolom, typeId = "t", typeName = "Kolom", typeColor = "#039be5",
        date = LocalDate.parse(date), time = time, host = "Umboh Riedel", location = "",
        leader = "", mc = "", note = "", start = "${date}T$time:00+08:00", end = "",
    )

    private fun member(name: String, month: Int?, day: Int?, kolom: Int = 11) = Member(
        id = name, familyId = "f", name = name, familyName = "X", kolom = kolom, gender = "",
        maritalStatus = "", baptized = null, confirmed = null, category = "", flags = emptyList(),
        birthMonth = month, birthDay = day,
    )

    @Test
    fun birthdaysOnACalendarGrid() {
        val members = listOf(
            member("Budi", 2, 14), member("Ani", 2, 14), member("Leap", 2, 29),
            member("Aug", 8, 31), member("Oct", 10, 2), member("Nov", 11, 1), member("None", null, null),
        )
        val feb = birthdaysOn(members, monthGrid(java.time.YearMonth.of(2026, 2))) // 1–28 Feb 2026
        assertEquals(listOf("Ani", "Budi"), feb[LocalDate.of(2026, 2, 14)]!!.map { it.name })
        assertEquals(listOf("Leap"), feb[LocalDate.of(2026, 2, 28)]!!.map { it.name }) // 2026 is not a leap year
        assertEquals(2, feb.size)
        // September 2026's grid runs from Sunday 30 August to Saturday 3 October
        val sep = birthdaysOn(members, monthGrid(java.time.YearMonth.of(2026, 9)))
        assertEquals(listOf("Aug"), sep[LocalDate.of(2026, 8, 31)]!!.map { it.name })
        assertEquals(listOf("Oct"), sep[LocalDate.of(2026, 10, 2)]!!.map { it.name })
        assertNull(sep[LocalDate.of(2026, 11, 1)])
        assertEquals(2, sep.size)
    }

    @Test
    fun upcomingSplitsTodayWeekAndNext() {
        val today = LocalDate.of(2026, 10, 20)
        val up = upcomingServices(
            listOf(
                service("later", "2026-11-04"),
                service("week-end", "2026-10-26"), // today + 6: still this week
                service("today-late", "2026-10-20", "19:00"),
                service("today-early", "2026-10-20", "06:00"),
                service("tomorrow", "2026-10-21"),
                service("past", "2026-10-19"),
                service("after-week", "2026-10-27"),
            ),
            today,
        )
        assertEquals(listOf("today-early", "today-late"), up.today.map { it.id })
        assertEquals(listOf("tomorrow", "week-end"), up.week.map { it.id })
        assertEquals("after-week", up.next?.id)
    }

    @Test
    fun birthdaysTodayAndLeapDay() {
        val members = listOf(member("Budi", 10, 20), member("Ani", 10, 20), member("Leap", 2, 29), member("None", null, null))
        assertEquals(listOf("Ani", "Budi"), birthdaysToday(members, LocalDate.of(2026, 10, 20)).map { it.name })
        // 29 February is celebrated on 28 February in non-leap years
        assertEquals(listOf("Leap"), birthdaysToday(members, LocalDate.of(2026, 2, 28)).map { it.name })
        assertTrue(birthdaysToday(members, LocalDate.of(2028, 2, 28)).isEmpty())
        assertEquals(listOf("Leap"), birthdaysToday(members, LocalDate.of(2028, 2, 29)).map { it.name })
    }

    @Test
    fun upcomingBirthdaysWrapTheYear() {
        val members = listOf(member("NewYear", 1, 2), member("Tomorrow", 12, 31), member("Today", 12, 30))
        val next = upcomingBirthdays(members, LocalDate.of(2026, 12, 30))
        assertEquals(listOf("Tomorrow" to 1L, "NewYear" to 3L), next.map { it.first.name to it.second })
    }

    @Test
    fun kolomFilterAndFormatting() {
        val list = listOf(service("a", "2026-10-21", kolom = 11), service("b", "2026-10-21", kolom = 5))
        assertEquals(listOf("a"), list.inKolom(11) { it.kolom }.map { it.id })
        assertEquals(2, list.inKolom(null) { it.kolom }.size)
        assertEquals("19.00 – 20.30", timeRange("19:00"))
        assertEquals("23.30 – 01.00", timeRange("23:30"))
        assertEquals("Rabu, 21 Oktober 2026", longDate(LocalDate.of(2026, 10, 21)))
        assertEquals("Min, 25 Okt", shortDate(LocalDate.of(2026, 10, 25)))
        assertNull(upcomingServices(emptyList(), LocalDate.of(2026, 1, 1)).next)
    }

    @Test
    fun monthGridCoversWholeWeeksFromSunday() {
        // October 2026 starts on a Thursday and needs 5 weeks
        val oct = monthGrid(java.time.YearMonth.of(2026, 10))
        assertEquals(35, oct.size)
        assertEquals(LocalDate.of(2026, 9, 27), oct.first()) // a Sunday
        assertEquals(LocalDate.of(2026, 10, 31), oct.last())
        // February 2026 starts on a Sunday: exactly 4 weeks
        assertEquals(28, monthGrid(java.time.YearMonth.of(2026, 2)).size)
        // August 2026 starts on a Saturday and has 31 days: 6 weeks
        assertEquals(42, monthGrid(java.time.YearMonth.of(2026, 8)).size)
    }

    @Test
    fun congregationServicesShowInEveryKolom() {
        val all = listOf(service("k11", "2026-10-22", kolom = 11), service("k3", "2026-10-22", kolom = 3), service("pemuda", "2026-10-24", kolom = 0))
        assertEquals(listOf("k11", "pemuda"), all.forKolom(11).map { it.id })
        assertEquals(3, all.forKolom(null).size)
        assertEquals("Semua kolom", all[2].kolomText)
        assertEquals("Kolom 11", all[0].kolomText)
    }
}
