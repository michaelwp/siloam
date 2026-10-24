package id.gmimsiloam.jemaat.notify

import android.content.Context
import android.graphics.Typeface
import android.text.SpannableString
import android.text.SpannableStringBuilder
import android.text.style.StyleSpan
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.data.Member
import id.gmimsiloam.jemaat.data.NOTIFY_AHEAD_DAYS
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Repository
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.birthdaysToday
import id.gmimsiloam.jemaat.data.byKolom
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.dayLabel
import id.gmimsiloam.jemaat.data.dayPhrase
import id.gmimsiloam.jemaat.data.kolomHeading
import id.gmimsiloam.jemaat.data.kolomText
import id.gmimsiloam.jemaat.data.longDate
import id.gmimsiloam.jemaat.data.upcomingBirthdays
import id.gmimsiloam.jemaat.data.upcomingServices
import java.time.LocalDate
import java.time.LocalTime
import java.time.OffsetDateTime
import java.util.concurrent.TimeUnit

/** Tag of the queued service reminders (cancelled together on settings changes). */
private const val REMINDER_TAG = "service-reminder"

/** Data older than this is fetched again by the background check. */
private const val STALE_AFTER_MS = 6 * 60 * 60 * 1000L

/**
 * Runs about every hour in the background (also after a reboot: WorkManager
 * restores it). Keeps the data fresh, shows the daily summary once a day after
 * the chosen time, and queues reminders for today's services.
 */
class DailyWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {
    override suspend fun doWork(): Result {
        val ctx = applicationContext
        val prefs = Prefs(ctx)
        val repo = Repository(ctx)
        val cached = repo.cached()
        val today = LocalDate.now()
        val stale = cached == null || System.currentTimeMillis() - cached.fetchedAt > STALE_AFTER_MS ||
            !java.time.Instant.ofEpochMilli(cached.fetchedAt).atZone(java.time.ZoneId.systemDefault()).toLocalDate().isEqual(today)
        val data = if (stale) repo.refreshOrCached() else cached
        // the update check does not need the data: do it first
        repo.checkRelease()
        notifyUpdate(ctx, prefs)
        if (data == null) return Result.retry() // nothing to tell yet; try again later

        if (LocalTime.now().toSecondOfDay() / 60 >= prefs.dailyAtMinutes) {
            showDaily(ctx, prefs, data, today, force = false)
        }
        scheduleReminders(ctx, prefs, data, today)
        return Result.success()
    }

    companion object {
        private const val WORK = "daily-check"

        /** (Re)starts the hourly background check; call at app start and after settings change. */
        fun schedule(context: Context) {
            val req = PeriodicWorkRequestBuilder<DailyWorker>(1, TimeUnit.HOURS)
                .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.NOT_REQUIRED).build())
                .build()
            WorkManager.getInstance(context).enqueueUniquePeriodicWork(WORK, ExistingPeriodicWorkPolicy.UPDATE, req)
        }

        /**
         * Drops the reminders queued for today and runs the check, which queues
         * them again: call after changing the reminder time, the notification
         * filter, or the services switch (queued reminders keep the delay they were made with).
         */
        fun rescheduleReminders(context: Context) {
            WorkManager.getInstance(context).cancelAllWorkByTag(REMINDER_TAG)
            runNow(context)
        }

        /** Runs the check now (e.g. after changing the daily time). */
        fun runNow(context: Context) {
            WorkManager.getInstance(context)
                .enqueueUniqueWork("$WORK-now", ExistingWorkPolicy.REPLACE, OneTimeWorkRequestBuilder<DailyWorker>().build())
        }
    }
}

/**
 * The daily summary: the services and birthdays of today and of the next
 * [NOTIFY_AHEAD_DAYS] days (per the notification filter, grouped per kolom),
 * so each one is announced every day for a week before the day. Each kind at
 * most once a day; a kind counts as done only when it could be shown
 * (notifications allowed and switched on), so granting permission or
 * switching it on later the same day still brings it. [force] shows it again
 * (the "show now" button) and also counts as today's.
 */
internal fun showDaily(ctx: Context, prefs: Prefs, data: Snapshot, today: LocalDate, force: Boolean) {
    if (!Notifier.canPost(ctx)) return
    val day = today.toString()
    val filter = prefs.filter
    if (prefs.notifyServices && (force || prefs.lastDailyServices != day)) {
        prefs.lastDailyServices = day
        val up = upcomingServices(filter.services(data), today, NOTIFY_AHEAD_DAYS)
        val todays = up.today
        val ahead = up.week
        val all = todays + ahead
        if (all.isNotEmpty()) {
            val first = all.first()
            val at = { s: Service -> if (s.date == today) clock(s.time) else "${dayLabel(s.date, today)} ${clock(s.time)}" }
            Notifier.post(
                ctx,
                Notifier.ID_DAILY_SERVICES,
                Notifier.CHANNEL_SERVICES,
                title = when {
                    all.size == 1 && todays.isNotEmpty() -> "Ibadah hari ini, ${clock(first.time)} WITA"
                    all.size == 1 -> "Ibadah ${dayPhrase(first.date, today)}, ${clock(first.time)} WITA"
                    todays.isEmpty() -> "${ahead.size} ibadah dalam $NOTIFY_AHEAD_DAYS hari ke depan"
                    ahead.isEmpty() -> "${todays.size} ibadah hari ini"
                    else -> "${todays.size} ibadah hari ini, ${ahead.size} dalam $NOTIFY_AHEAD_DAYS hari ke depan"
                },
                text = all.byKolom { it.kolom }.joinToString(" · ") { (k, list) ->
                    "${kolomHeading(k)}: " + list.joinToString(", ") { "${at(it)} ${it.fullTitle}" }
                },
                lines = grouped(all, { it.kolom }) { serviceLine(it, at(it)) },
                tab = "home",
            )
        }
    }
    if (prefs.notifyBirthdays && (force || prefs.lastDailyBirthdays != day)) {
        prefs.lastDailyBirthdays = day
        val members = filter.members(data)
        val todays = birthdaysToday(members, today)
        val ahead = upcomingBirthdays(members, today, NOTIFY_AHEAD_DAYS)
        if (todays.isNotEmpty() || ahead.isNotEmpty()) {
            val byDay = listOf(today to todays) +
                ahead.groupBy({ today.plusDays(it.second) }, { it.first }).toList()
            val lines = byDay.filter { it.second.isNotEmpty() }.flatMap { (d, list) -> birthdayLines(list, dayLabel(d, today)) }
            val title = when {
                todays.size == 1 -> "🎂 ${todays.first().name} berulang tahun hari ini"
                todays.isNotEmpty() -> "🎂 ${todays.size} anggota jemaat berulang tahun hari ini"
                ahead.size == 1 -> "🎂 ${ahead.first().first.name} berulang tahun ${dayPhrase(today.plusDays(ahead.first().second), today)}"
                else -> "🎂 ${ahead.size} anggota jemaat berulang tahun dalam $NOTIFY_AHEAD_DAYS hari ke depan"
            }
            Notifier.post(
                ctx,
                Notifier.ID_DAILY_BIRTHDAYS,
                Notifier.CHANNEL_BIRTHDAYS,
                title = title,
                text = lines.joinToString(" · ") + if (todays.isNotEmpty()) ". Selamat ulang tahun!" else "",
                lines = lines,
                tab = "birthdays",
            )
        }
    }
}

/**
 * "A new version is available", once per version, and only counted as shown
 * when it could be shown (as the daily summary).
 */
private fun notifyUpdate(ctx: Context, prefs: Prefs) {
    val update = prefs.availableUpdate(BuildConfig.VERSION_NAME) ?: return
    if (prefs.notifiedUpdate == update.latestVersion || !Notifier.canPost(ctx)) return
    Notifier.postUpdate(ctx, update.latestVersion, update.downloadUrl)
    prefs.notifiedUpdate = update.latestVersion
}

/** A bold "Kolom 11:" for the expanded notification. */
private fun heading(kolom: Int): CharSequence =
    SpannableString("${kolomHeading(kolom)}:").apply { setSpan(StyleSpan(Typeface.BOLD), 0, length, 0) }

/**
 * The expanded notification's lines, per kolom ("Semua kolom" first): a bold
 * "Kolom 11:" heading, then "- 19.00 · Ibadah Kolom · …" for each item.
 */
private fun <T> grouped(items: List<T>, kolomOf: (T) -> Int, line: (T) -> String): List<CharSequence> =
    items.byKolom(kolomOf).flatMap { (k, list) -> listOf(heading(k)) + list.map { "- " + line(it) } }

/** One line per kolom: "Besok · Kolom 11: Budi Umboh, Erwin Umboh" (the day and kolom in bold). */
private fun birthdayLines(birthdays: List<Member>, day: String): List<CharSequence> =
    birthdays.byKolom { it.kolom }.map { (k, list) ->
        val head = SpannableString("$day · ${kolomHeading(k)}:").apply { setSpan(StyleSpan(Typeface.BOLD), 0, length, 0) }
        SpannableStringBuilder(head).append(" ").append(list.joinToString(", ") { it.name })
    }

/** "19.00 · Ibadah Kolom · …", or "Besok 19.00 · …" for a later day ([at]). */
private fun serviceLine(s: Service, at: String) = buildString {
    append("$at · ${s.fullTitle}")
    if (s.leader.isNotEmpty()) append(" · Khadim: ${s.leader}")
}

/** Queues a reminder [Prefs.remindBeforeMinutes] before each of today's services that is still ahead. */
private fun scheduleReminders(ctx: Context, prefs: Prefs, data: Snapshot, today: LocalDate) {
    val before = prefs.remindBeforeMinutes
    if (!prefs.notifyServices || before <= 0) return
    // forget reminders of other days
    val todays = upcomingServices(prefs.filter.services(data), today).today
    prefs.remindedServices = prefs.remindedServices.filter { id -> todays.any { it.id == id } }.toSet()
    val wm = WorkManager.getInstance(ctx)
    for (s in todays) {
        if (s.id in prefs.remindedServices) continue
        val start = runCatching { OffsetDateTime.parse(s.start).toInstant().toEpochMilli() }.getOrNull() ?: continue
        val delay = start - before * 60_000L - System.currentTimeMillis()
        if (start <= System.currentTimeMillis()) continue // already started
        val req = OneTimeWorkRequestBuilder<ReminderWorker>()
            .addTag(REMINDER_TAG)
            .setInitialDelay(delay.coerceAtLeast(0), TimeUnit.MILLISECONDS)
            .setInputData(
                workDataOf(
                    "id" to s.id,
                    "start" to start,
                    "title" to "Ibadah dimulai ${clock(s.time)} WITA",
                    "text" to "${s.kolomText} · ${s.fullTitle}" + if (s.place.isNotEmpty()) " · ${s.place}" else "",
                    "detail" to listOfNotNull(
                        "${s.kolomText} · ${s.fullTitle}",
                        s.place.ifEmpty { null },
                        s.leader.ifEmpty { null }?.let { "Khadim: $it" },
                        s.mc.ifEmpty { null }?.let { "Pembawa acara: $it" },
                        longDate(s.date),
                    ).joinToString("\n"),
                ),
            )
            .build()
        // KEEP: the hourly check must not push an already queued reminder back
        wm.enqueueUniqueWork("remind-${s.id}", ExistingWorkPolicy.KEEP, req)
    }
}

/** Posts one service reminder (queued by [DailyWorker]). */
class ReminderWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {
    override suspend fun doWork(): Result {
        val prefs = Prefs(applicationContext)
        val id = inputData.getString("id") ?: return Result.success()
        val start = inputData.getLong("start", 0)
        // the filter may have changed since the reminder was queued, or the
        // service been removed; and Android may run it late (Doze): never
        // remind about a started service
        val cached = Repository(applicationContext).cached()
        val wanted = cached == null || prefs.filter.services(cached).any { it.id == id }
        val started = start > 0 && System.currentTimeMillis() >= start
        if (!prefs.notifyServices || id in prefs.remindedServices || started || !wanted) {
            return Result.success()
        }
        Notifier.post(
            applicationContext,
            id.hashCode(),
            Notifier.CHANNEL_SERVICES,
            title = inputData.getString("title") ?: "Ibadah segera dimulai",
            text = inputData.getString("text") ?: "",
            lines = (inputData.getString("detail") ?: "").split("\n"),
            tab = "schedule",
        )
        prefs.remindedServices = prefs.remindedServices + id
        return Result.success()
    }
}
