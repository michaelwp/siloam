package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.LocalDate

class ParseTest {
    @Test
    fun parsesThePublicViews() {
        val services = Parse.services(
            """[{"id":"s1","kolom":11,"typeId":"t1","date":"2026-01-21T00:00:00Z","time":"19:00","host":"Samola Ngantung",
               "location":"","leader":"Pdt, Eva","leaderId":null,"mc":"Dkn Erwin","mcId":"m1","note":"","typeName":"Kolom",
               "typeColor":"#039be5","start":"2026-01-21T19:00:00+08:00","end":"2026-01-21T20:30:00+08:00"}]""",
        )
        val s = services.single()
        assertEquals(LocalDate.of(2026, 1, 21), s.date)
        assertEquals("Kel. Samola Ngantung", s.title)
        assertEquals("Ibadah Kolom · Kel. Samola Ngantung", s.fullTitle)
        assertEquals("Rumah Kel. Samola Ngantung", s.place)

        // the public member view: birth year, age, birth place are absent
        val m = Parse.members(
            """[{"id":"m1","familyId":"f1","order":1,"name":"Danny","birthPlace":"","birthDate":null,"gender":"Laki Laki",
               "maritalStatus":"Menikah","relationship":"","baptized":true,"confirmed":null,"familyName":"Najoan Ismawi",
               "kolom":11,"age":null,"ageText":"","category":"PKB","elderly":false,"birthMonth":11,"birthDay":13}]""",
        ).single()
        assertEquals(true, m.baptized)
        assertNull(m.confirmed)
        assertEquals(11, m.birthMonth)

        val f = Parse.families("""[{"id":"f1","kolom":11,"no":3,"name":"Najoan","note":"","memberCount":4,"weddingMonth":null,"weddingDay":null}]""").single()
        assertNull(f.weddingMonth)
        assertEquals(4, f.memberCount)
    }
}
