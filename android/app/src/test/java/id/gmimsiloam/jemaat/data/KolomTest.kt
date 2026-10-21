package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class KolomTest {
    private fun snapshot(vararg koloms: Int) = Snapshot(
        koloms = koloms.map { Kolom("k$it", it) }, types = emptyList(), families = emptyList(),
        members = emptyList(), services = emptyList(), categories = DEFAULT_CATEGORIES, fetchedAt = 0,
    )

    @Test
    fun deletedKolomFallsBackToAll() {
        assertEquals(11, snapshot(1, 11).effectiveKolom(11))
        assertNull(snapshot(1, 2).effectiveKolom(11)) // kolom 11 was deleted on the server
        assertNull(snapshot(1, 11).effectiveKolom(null))
        // a server from before koloms existed lists none: keep the choice
        assertEquals(11, snapshot().effectiveKolom(11))
    }
}
