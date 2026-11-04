package id.gmimsiloam.jemaat.data

// Password rules shown live while typing. Keep in sync with web/src/password.ts
// and internal/auth/password.go, which enforces the same rules on the server.

const val MIN_PASSWORD_LENGTH = 10

/** bcrypt's limit, in UTF-8 bytes (accented letters and emoji take 2–4 bytes) */
const val MAX_PASSWORD_BYTES = 72

private val COMMON_WORDS = setOf(
    "password", "passwrd", "pass", "admin", "administrator",
    "qwerty", "qwertyuiop", "asdf", "asdfgh", "zxcvbn",
    "letmein", "welcome", "iloveyou", "monkey", "dragon",
    "sunshine", "princess", "football", "master", "secret",
    "rahasia", "sandi", "katasandi", "indonesia", "manado",
    "jemaat", "kolom", "gereja", "siloam", "tuhan",
    "yesus", "jesus", "haleluya", "amin",
)

data class PasswordRule(val id: String, val label: String, val ok: Boolean)

private fun codePoints(p: String) = p.codePoints().toArray().toList()

/** The rules and whether [password] meets each (the "max" rule is shown only when broken). */
fun passwordRules(password: String, username: String = ""): List<PasswordRule> {
    val cps = codePoints(password)
    val user = username.trim().lowercase()
    val letters = cps.filter { Character.isLetter(it) }.joinToString("") { String(Character.toChars(Character.toLowerCase(it))) }
    return listOf(
        PasswordRule("length", "Minimal $MIN_PASSWORD_LENGTH karakter", cps.size >= MIN_PASSWORD_LENGTH),
        PasswordRule("case", "Huruf besar dan huruf kecil", cps.any { Character.isUpperCase(it) } && cps.any { Character.isLowerCase(it) }),
        PasswordRule("digit", "Minimal satu angka", cps.any { Character.isDigit(it) }),
        PasswordRule(
            "symbol",
            "Minimal satu simbol (mis. ! @ # ?)",
            cps.any { !Character.isLetter(it) && !Character.isDigit(it) && !Character.isWhitespace(it) },
        ),
        PasswordRule("username", "Tidak mengandung username", user.length < 3 || !password.lowercase().contains(user)),
        PasswordRule("common", "Bukan password yang umum dipakai", letters !in COMMON_WORDS),
        PasswordRule(
            "max",
            "Maksimal $MAX_PASSWORD_BYTES byte (huruf beraksen dan emoji dihitung lebih dari satu)",
            password.toByteArray(Charsets.UTF_8).size <= MAX_PASSWORD_BYTES,
        ),
    )
}

fun passwordValid(password: String, username: String = "") = passwordRules(password, username).all { it.ok }
