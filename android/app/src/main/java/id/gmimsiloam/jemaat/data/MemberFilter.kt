package id.gmimsiloam.jemaat.data

/** A yes/no/unknown field (baptis, sidi) as filtered on the Jemaat tab; [ANY] = no filter. */
enum class Tri(val key: String) {
    ANY(""), YES("yes"), NO("no"), UNKNOWN("unknown");

    fun matches(v: Boolean?) = when (this) {
        ANY -> true
        YES -> v == true
        NO -> v == false
        UNKNOWN -> v == null
    }

    companion object {
        fun of(key: String) = entries.firstOrNull { it.key == key } ?: ANY
    }
}

/** The Jemaat tab's filters, as on the web's Daftar Anggota Jemaat: baptis, sidi, marital status, gender. */
data class MemberFilter(
    val baptized: Tri = Tri.ANY,
    val confirmed: Tri = Tri.ANY,
    val status: String = "",
    val gender: String = "",
) {
    val active get() = baptized != Tri.ANY || confirmed != Tri.ANY || status.isNotEmpty() || gender.isNotEmpty()

    fun matches(m: Member) = baptized.matches(m.baptized) && confirmed.matches(m.confirmed) &&
        (status.isEmpty() || m.maritalStatus == status) && (gender.isEmpty() || m.gender == gender)

    /** For rememberSaveable: "yes|unknown|Menikah|Perempuan". */
    fun encode() = "${baptized.key}|${confirmed.key}|$status|$gender"

    companion object {
        val none = MemberFilter()

        fun decode(s: String) = s.split('|', limit = 4).let { p ->
            MemberFilter(Tri.of(p.getOrElse(0) { "" }), Tri.of(p.getOrElse(1) { "" }), p.getOrElse(2) { "" }, p.getOrElse(3) { "" })
        }
    }
}

/** The web's marital statuses, in its order. */
val MARITAL_STATUSES = listOf(
    "Belum Menikah",
    "Menikah",
    "Cerai Hidup / Duda",
    "Cerai Hidup / Janda",
    "Cerai Mati / Duda",
    "Cerai Mati / Janda",
)

/** The statuses to offer: the known ones, then any other the data has. */
fun maritalStatuses(members: List<Member>): List<String> {
    val used = members.map { it.maritalStatus }.filter { it.isNotEmpty() }.toSet()
    return MARITAL_STATUSES + (used - MARITAL_STATUSES.toSet()).sorted()
}
