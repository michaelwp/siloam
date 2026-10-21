package id.gmimsiloam.jemaat.ui

import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class ServerUrlTest {
    @Test
    fun releaseNeedsHttps() {
        assertNull(serverUrlError("https://siloam.fly.dev", debug = false))
        assertNotNull(serverUrlError("http://192.168.1.5:8080", debug = false))
        assertNotNull(serverUrlError("siloam.fly.dev", debug = false))
        assertNotNull(serverUrlError("https://", debug = false))
        assertNotNull(serverUrlError("", debug = false))
    }

    @Test
    fun debugAllowsHttp() {
        assertNull(serverUrlError("http://10.0.2.2:8080", debug = true))
    }
}
