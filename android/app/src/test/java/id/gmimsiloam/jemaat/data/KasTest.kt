package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Test

class KasTest {
    @Test
    fun rupiahLikeTheWeb() {
        assertEquals("Rp 0", rupiah(0))
        assertEquals("Rp 950", rupiah(950))
        assertEquals("Rp 1.250.000", rupiah(1_250_000))
        assertEquals("-Rp 300.000", rupiah(-300_000))
    }

    @Test
    fun parsesTheSummary() {
        val s = parseKas("""[{"kolom":3,"in":1500000,"out":300000,"count":4,"last":"2026-11-02T00:00:00Z","month":"2026-11","monthIn":500000,"monthOut":0}]""")
        assertEquals(listOf(KasSummary(3, 1_500_000, 300_000, "2026-11", 500_000, 0)), s)
        assertEquals(1_200_000, s[0].balance)
        assertEquals("November 2026", monthLabel(s[0].month))
    }
}
