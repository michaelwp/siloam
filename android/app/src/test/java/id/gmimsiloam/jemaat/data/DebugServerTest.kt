package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DebugServerTest {
    @Test
    fun acceptsHttpAddresses() {
        assertEquals("http://localhost:8080", debugServerUrl(" http://localhost:8080/ "))
        assertEquals("http://192.168.1.5:8080", debugServerUrl("http://192.168.1.5:8080"))
        assertEquals("https://siloam.fly.dev", debugServerUrl("https://siloam.fly.dev"))
    }

    @Test
    fun refusesOthers() {
        assertNull(debugServerUrl(""))
        assertNull(debugServerUrl("localhost:8080"))
        assertNull(debugServerUrl("ftp://host"))
        assertNull(debugServerUrl("http://"))
        assertNull(debugServerUrl("http://host/?a=1"))
    }
}
