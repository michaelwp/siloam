package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Test

class PublicStatsTest {
    @Test
    fun parsesTheVisitorsSummary() {
        val json = """
            {"families":2,"members":3,"male":2,"female":1,
             "baptized":{"yes":1,"no":0,"unknown":2},"confirmed":{"yes":0,"no":1,"unknown":2},
             "maritalStatuses":[{"label":"Belum Menikah","value":1},{"label":"Menikah","value":2}],
             "categories":[
               {"id":"a","key":"Pemuda","name":"Pemuda","kind":"main","minAge":17,"maxAge":30,"count":1,"male":1,"female":0},
               {"id":"b","key":"Lansia","name":"Lansia","kind":"flag","minAge":60,"maxAge":null,"count":0,"male":0,"female":0}],
             "birthMonths":[0,0,1,0,0,0,0,0,0,0,0,0],"familySizes":[1,1]}
        """.trimIndent()
        val p = parsePublicStats(json)
        assertEquals(3, p.stats.members)
        assertEquals(listOf("Belum Menikah" to 1, "Menikah" to 2), p.stats.byStatus)
        assertEquals(1, p.stats.bipra["Pemuda"])
        assertEquals("17 – 30", p.categories[0].range)
        assertEquals(1, p.stats.bipraByGender.size) // flags are not in the gender chart
        assertEquals(1, p.stats.birthdaysPerMonth[2])
        assertEquals(listOf(1, 1), p.stats.familySizes)
        assertEquals(YesNo(1, 0, 2), p.stats.baptized)
    }
}
