package id.gmimsiloam.jemaat.data

import org.json.JSONArray
import org.json.JSONObject
import java.time.LocalDate

// Data as the server's public (visitor) view returns it: no birth years, ages,
// birth places, family relationships or addresses.

data class Kolom(val id: String, val number: Int)

data class WorshipType(val id: String, val name: String, val color: String)

data class Family(
    val id: String,
    val kolom: Int,
    val no: Int,
    val name: String,
    val note: String,
    val memberCount: Int,
    val weddingMonth: Int?,
    val weddingDay: Int?,
)

data class Member(
    val id: String,
    val familyId: String,
    val name: String,
    val familyName: String,
    val kolom: Int,
    val gender: String,
    val maritalStatus: String,
    val baptized: Boolean?,
    val confirmed: Boolean?,
    /** the main BIPRA category's key ("" = none) */
    val category: String,
    /** keys of the BIPRA flags, e.g. ["Lansia"] */
    val flags: List<String>,
    val birthMonth: Int?,
    val birthDay: Int?,
)

data class Service(
    val id: String,
    val kolom: Int,
    val typeId: String,
    val typeName: String,
    val typeColor: String,
    /** the local date of the service (the server sends midnight UTC of it) */
    val date: LocalDate,
    /** "19:00", local time (WITA) */
    val time: String,
    val host: String,
    val location: String,
    val leader: String,
    val mc: String,
    val note: String,
    /** absolute start/end, ISO-8601 with offset */
    val start: String,
    val end: String,
) {
    /** "Kel. Umboh Riedel", or "Ibadah Pemuda" without a host */
    val title: String get() = if (host.isNotEmpty()) "Kel. $host" else "Ibadah $typeName"
    val fullTitle: String get() = "Ibadah $typeName" + if (host.isNotEmpty()) " · Kel. $host" else ""
    val place: String get() = location.ifEmpty { if (host.isNotEmpty()) "Rumah Kel. $host" else "" }
}

/** Everything the app shows, as fetched at [fetchedAt] (epoch millis). */
data class Snapshot(
    val koloms: List<Kolom>,
    val types: List<WorshipType>,
    val families: List<Family>,
    val members: List<Member>,
    val services: List<Service>,
    /** the Kategorial BIPRA categories, in order */
    val categories: List<Category>,
    val fetchedAt: Long,
)

private fun JSONObject.str(key: String) = if (isNull(key)) "" else optString(key, "")
private fun JSONObject.intOrNull(key: String) = if (has(key) && !isNull(key)) optInt(key) else null
private fun JSONObject.boolOrNull(key: String) = if (has(key) && !isNull(key)) optBoolean(key) else null

private inline fun <T> JSONArray.mapObjects(f: (JSONObject) -> T): List<T> =
    (0 until length()).map { f(getJSONObject(it)) }

object Parse {
    fun koloms(json: String) = JSONArray(json).mapObjects { Kolom(it.str("id"), it.optInt("number")) }

    fun types(json: String) = JSONArray(json).mapObjects { WorshipType(it.str("id"), it.str("name"), it.str("color")) }

    fun families(json: String) = JSONArray(json).mapObjects {
        Family(
            id = it.str("id"),
            kolom = it.optInt("kolom"),
            no = it.optInt("no"),
            name = it.str("name"),
            note = it.str("note"),
            memberCount = it.optInt("memberCount"),
            weddingMonth = it.intOrNull("weddingMonth"),
            weddingDay = it.intOrNull("weddingDay"),
        )
    }

    fun members(json: String) = JSONArray(json).mapObjects {
        Member(
            id = it.str("id"),
            familyId = it.str("familyId"),
            name = it.str("name"),
            familyName = it.str("familyName"),
            kolom = it.optInt("kolom"),
            gender = it.str("gender"),
            maritalStatus = it.str("maritalStatus"),
            baptized = it.boolOrNull("baptized"),
            confirmed = it.boolOrNull("confirmed"),
            category = it.str("category"),
            // servers before editable categories send only "elderly" (Lansia)
            flags = it.optJSONArray("flags")?.let { a -> (0 until a.length()).map { i -> a.getString(i) } }
                ?: if (it.optBoolean("elderly")) listOf(LANSIA) else emptyList(),
            birthMonth = it.intOrNull("birthMonth"),
            birthDay = it.intOrNull("birthDay"),
        )
    }

    fun categories(json: String) = JSONArray(json).mapObjects {
        Category(
            key = it.str("key"),
            label = it.str("name"),
            range = bipraRange(it.intOrNull("minAge"), it.intOrNull("maxAge"), it.optBoolean("orMarried") || it.str("marital") == "married"),
            isFlag = it.str("kind") == "flag",
        )
    }

    fun services(json: String) = JSONArray(json).mapObjects {
        Service(
            id = it.str("id"),
            kolom = it.optInt("kolom"),
            typeId = it.str("typeId"),
            typeName = it.str("typeName"),
            typeColor = it.str("typeColor"),
            date = LocalDate.parse(it.str("date").take(10)),
            time = it.str("time"),
            host = it.str("host"),
            location = it.str("location"),
            leader = it.str("leader"),
            mc = it.str("mc"),
            note = it.str("note"),
            start = it.str("start"),
            end = it.str("end"),
        )
    }
}
