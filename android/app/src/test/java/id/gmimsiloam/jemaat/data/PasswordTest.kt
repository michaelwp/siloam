package id.gmimsiloam.jemaat.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PasswordTest {
    private fun broken(p: String, user: String = "") = passwordRules(p, user).filterNot { it.ok }.map { it.id }

    @Test
    fun rulesLikeTheServer() {
        assertTrue(passwordValid("Kuat!Sekali9", "budi"))
        assertEquals(listOf("length", "case", "digit", "symbol"), broken("abc"))
        assertEquals(listOf("username"), broken("Budi!12345x", "budi"))
        assertEquals(listOf("common"), broken("Siloam!2024", ""))
        assertFalse(passwordValid("A1!" + "é".repeat(40)))
    }
}
