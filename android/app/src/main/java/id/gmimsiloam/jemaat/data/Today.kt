package id.gmimsiloam.jemaat.data

import java.time.LocalDate
import java.time.MonthDay
import java.time.Year

/** Services shown as "this week": the days after today, up to this many (as on the web). */
const val WEEK_AHEAD_DAYS = 6L

val MONTHS = listOf(
    "Januari", "Februari", "Maret", "April", "Mei", "Juni",
    "Juli", "Agustus", "September", "Oktober", "November", "Desember",
)
/** Indexed by java.time DayOfWeek.value % 7 (Minggu first). */
val WEEKDAYS = listOf("Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu")

fun weekday(d: LocalDate) = WEEKDAYS[d.dayOfWeek.value % 7]

/** "Rabu, 21 Oktober 2026" */
fun longDate(d: LocalDate) = "${weekday(d)}, ${d.dayOfMonth} ${MONTHS[d.monthValue - 1]} ${d.year}"

/** "Rab, 21 Okt" */
fun shortDate(d: LocalDate) = "${weekday(d).take(3)}, ${d.dayOfMonth} ${MONTHS[d.monthValue - 1].take(3)}"

/** "19:00" → "19.00" (Indonesian style) */
fun clock(t: String) = t.replace(':', '.')

/** "19.00 – 20.30" (services are assumed to last 90 minutes, as on the web) */
fun timeRange(t: String): String {
    val parts = t.split(':')
    val h = parts.getOrNull(0)?.toIntOrNull() ?: return clock(t)
    val m = parts.getOrNull(1)?.toIntOrNull() ?: 0
    val end = h * 60 + m + 90
    return "${clock(t)} – %02d.%02d".format((end / 60) % 24, end % 60)
}

/** Items of the viewed kolom (null = all). */
fun <T> List<T>.inKolom(kolom: Int?, of: (T) -> Int) = if (kolom == null) this else filter { of(it) == kolom }

/** Services of the viewed kolom, and those for the whole congregation (kolom 0, e.g. a Pemuda service). */
fun List<Service>.forKolom(kolom: Int?) = if (kolom == null) this else filter { it.kolom == kolom || it.kolom == 0 }

/** "Kolom 11", or "Semua kolom" for a service of the whole congregation. */
val Service.kolomText: String get() = if (kolom == 0) "Semua kolom" else "Kolom $kolom"

/**
 * The days shown in a month calendar: whole weeks from Sunday (as the
 * weekday names, Minggu first) covering [month], so 4 to 6 rows.
 */
fun monthGrid(month: java.time.YearMonth): List<LocalDate> {
    val first = month.atDay(1)
    val start = first.minusDays((first.dayOfWeek.value % 7).toLong())
    val weeks = (first.dayOfWeek.value % 7 + month.lengthOfMonth() + 6) / 7
    return (0 until weeks * 7).map { start.plusDays(it.toLong()) }
}

data class Upcoming(val today: List<Service>, val week: List<Service>, val next: Service?)

private val byStart = compareBy<Service>({ it.date }, { it.time })

/** Today's services, those of the next [WEEK_AHEAD_DAYS] days, and the first one after. */
fun upcomingServices(services: List<Service>, today: LocalDate): Upcoming {
    val sorted = services.sortedWith(byStart)
    val end = today.plusDays(WEEK_AHEAD_DAYS)
    return Upcoming(
        today = sorted.filter { it.date == today },
        week = sorted.filter { it.date.isAfter(today) && !it.date.isAfter(end) },
        next = sorted.firstOrNull { it.date.isAfter(end) },
    )
}

/**
 * Days from [today] until the next birthday on [month]/[day]; 29 February is
 * celebrated on 28 February in other years (as on the web).
 */
fun daysUntil(month: Int, day: Int, today: LocalDate): Long {
    fun on(year: Int): LocalDate {
        val d = if (month == 2 && day == 29 && !Year.isLeap(year.toLong())) 28 else day
        return LocalDate.of(year, month, d)
    }
    var next = on(today.year)
    if (next.isBefore(today)) next = on(today.year + 1)
    return java.time.temporal.ChronoUnit.DAYS.between(today, next)
}

fun Member.hasBirthday(): Boolean {
    val m = birthMonth ?: return false
    val d = birthDay ?: return false
    return runCatching { MonthDay.of(m, d) }.isSuccess
}

fun birthdaysToday(members: List<Member>, today: LocalDate) =
    members.filter { it.hasBirthday() && daysUntil(it.birthMonth!!, it.birthDay!!, today) == 0L }
        .sortedBy { it.name.lowercase() }

/** Birthdays of the next [days] days (not today), soonest first. */
fun upcomingBirthdays(members: List<Member>, today: LocalDate, days: Long = 7) =
    members.filter { it.hasBirthday() }
        .map { it to daysUntil(it.birthMonth!!, it.birthDay!!, today) }
        .filter { (_, d) -> d in 1..days }
        .sortedWith(compareBy({ it.second }, { it.first.name.lowercase() }))

fun anniversariesToday(families: List<Family>, today: LocalDate) =
    families.filter {
        it.weddingMonth != null && it.weddingDay != null &&
            daysUntil(it.weddingMonth, it.weddingDay, today) == 0L
    }

/**
 * The kolom to show: the chosen one while it still exists on the server, else
 * all koloms (as the web falls back when a remembered kolom was deleted).
 */
fun Snapshot.effectiveKolom(chosen: Int?): Int? =
    chosen?.takeIf { k -> koloms.isEmpty() || koloms.any { it.number == k } }
