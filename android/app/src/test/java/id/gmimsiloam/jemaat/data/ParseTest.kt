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

    @Test
    fun parsesCategoriesAndFlags() {
        val cats = Parse.categories(
            """[{"id":"1","key":"PKB","name":"PKB","kind":"main","order":4,"color":"#3f51b5","minAge":31,"maxAge":null,
                 "gender":"Laki Laki","marital":"","orMarried":true},
                {"id":"2","key":"Pemudi","name":"Pemudi Gereja","kind":"main","order":7,"color":"#8e24aa","minAge":17,"maxAge":30,
                 "gender":"Perempuan","marital":"unmarried","orMarried":false},
                {"id":"3","key":"Lansia","name":"Lansia","kind":"flag","order":6,"color":"#616161","minAge":60,"maxAge":null,
                 "gender":"","marital":"","orMarried":false}]""",
        )
        assertEquals(Category("PKB", "PKB", "31+ / menikah"), cats[0])
        assertEquals(Category("Pemudi", "Pemudi Gereja", "17 – 30"), cats[1])
        assertEquals(Category("Lansia", "Lansia", "60+", isFlag = true), cats[2])

        val base = """"id":"m1","familyId":"f1","name":"A","familyName":"X","kolom":11,"gender":"Perempuan","category":"WKI""""
        assertEquals(listOf("Lansia", "Baru"), Parse.members("""[{$base,"flags":["Lansia","Baru"],"elderly":true}]""").single().flags)
        // an older server: only "elderly"
        assertEquals(listOf(LANSIA), Parse.members("""[{$base,"elderly":true}]""").single().flags)
        assertEquals(emptyList<String>(), Parse.members("""[{$base,"flags":null,"elderly":false}]""").single().flags)
    }
}
