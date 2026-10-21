package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class UpdateTest {
    @Test
    fun comparesVersionsNumerically() {
        assertTrue(isNewer("1.0.1", "1.0.0"))
        assertTrue(isNewer("1.0.10", "1.0.9")) // not as text
        assertTrue(isNewer("2", "1.9.9"))
        assertTrue(isNewer("1.1", "1.0.5"))
        assertFalse(isNewer("1.0.0", "1.0.0"))
        assertFalse(isNewer("1.0", "1.0.0"))
        assertFalse(isNewer("1.0.0", "1.0.1"))
        assertFalse(isNewer("1.0.0", "1.0.0-debug"))
    }

    @Test
    fun readsTheServersAnswer() {
        assertEquals(
            AppRelease("1.0.1", "https://example.org/app.apk"),
            parseRelease("""{"latestVersion":"1.0.1","downloadUrl":"https://example.org/app.apk"}"""),
        )
        assertNull(parseRelease("""{"latestVersion":"","downloadUrl":""}""")) // not configured
        assertNull(parseRelease("""{"latestVersion":"1.0.1","downloadUrl":"http://example.org/app.apk"}"""))
    }
}
