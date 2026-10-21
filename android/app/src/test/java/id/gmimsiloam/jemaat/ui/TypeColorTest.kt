package id.gmimsiloam.jemaat.ui

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Test

class TypeColorTest {

    @Test
    fun lightColorsGetDarkTextDarkColorsWhite() {
        val dark = Color(0xDE000000)
        // the web's palette (web/src/schedule.ts TYPE_COLORS)
        assertEquals(dark, onTypeColor(Color(0xFFF6BF26))) // Kuning
        assertEquals(dark, onTypeColor(Color(0xFF33B679))) // Hijau Muda
        assertEquals(Color.White, onTypeColor(Color(0xFF3F51B5))) // Biru Tua
        assertEquals(Color.White, onTypeColor(Color(0xFF8E24AA))) // Ungu
        assertEquals(Color.White, onTypeColor(Color(0xFF616161))) // Abu-abu
        assertEquals(Color.White, onTypeColor(Color(0xFFD50000))) // Merah
    }
}
