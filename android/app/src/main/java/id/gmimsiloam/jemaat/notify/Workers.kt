package id.gmimsiloam.jemaat.notify

import android.content.Context
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
import id.gmimsiloam.jemaat.data.Prefs
import id.gmimsiloam.jemaat.data.Repository
import id.gmimsiloam.jemaat.data.Service
import id.gmimsiloam.jemaat.data.Snapshot
import id.gmimsiloam.jemaat.data.birthdaysToday
import id.gmimsiloam.jemaat.data.clock
import id.gmimsiloam.jemaat.data.effectiveKolom
import id.gmimsiloam.jemaat.data.inKolom
import id.gmimsiloam.jemaat.data.longDate
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
         * them again: call after changing the reminder time, the kolom, or the
         * services switch (queued reminders keep the delay they were made with).
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
 * The daily summary: today's services and today's birthdays (of the chosen
 * kolom), each at most once a day. A kind counts as done only when it could be
 * shown (notifications allowed and switched on), so granting permission or
 * switching it on later the same day still brings it. [force] shows it again
 * (the "show now" button) and also counts as today's.
 */
internal fun showDaily(ctx: Context, prefs: Prefs, data: Snapshot, today: LocalDate, force: Boolean) {
    if (!Notifier.canPost(ctx)) return
    val day = today.toString()
    val kolom = data.effectiveKolom(prefs.kolom)
    if (prefs.notifyServices && (force || prefs.lastDailyServices != day)) {
        prefs.lastDailyServices = day
        val todays = upcomingServices(data.services.inKolom(kolom) { it.kolom }, today).today
        if (todays.isNotEmpty()) {
            val first = todays.first()
            Notifier.post(
                ctx,
                Notifier.ID_DAILY_SERVICES,
                Notifier.CHANNEL_SERVICES,
                title = if (todays.size == 1) "Ibadah hari ini, ${clock(first.time)} WITA" else "${todays.size} ibadah hari ini",
                text = todays.joinToString(" · ") { "${clock(it.time)} ${it.fullTitle}" },
                lines = todays.map { serviceLine(it) },
                tab = "home",
            )
        }
    }
    if (prefs.notifyBirthdays && (force || prefs.lastDailyBirthdays != day)) {
        prefs.lastDailyBirthdays = day
        val birthdays = birthdaysToday(data.members.inKolom(kolom) { it.kolom }, today)
        if (birthdays.isNotEmpty()) {
            Notifier.post(
                ctx,
                Notifier.ID_DAILY_BIRTHDAYS,
                Notifier.CHANNEL_BIRTHDAYS,
                title = if (birthdays.size == 1) "🎂 ${birthdays.first().name} berulang tahun hari ini"
                else "🎂 ${birthdays.size} anggota jemaat berulang tahun hari ini",
                text = birthdays.joinToString(", ") { it.name } + ". Selamat ulang tahun!",
                lines = birthdays.map { "${it.name} · Kel. ${it.familyName}" },
                tab = "birthdays",
            )
        }
    }
}

private fun serviceLine(s: Service) = buildString {
    append("${clock(s.time)} · ${s.fullTitle}")
    if (s.leader.isNotEmpty()) append(" · Pemimpin: ${s.leader}")
}

/** Queues a reminder [Prefs.remindBeforeMinutes] before each of today's services that is still ahead. */
private fun scheduleReminders(ctx: Context, prefs: Prefs, data: Snapshot, today: LocalDate) {
    val before = prefs.remindBeforeMinutes
    if (!prefs.notifyServices || before <= 0) return
    // forget reminders of other days
    val todays = upcomingServices(data.services.inKolom(data.effectiveKolom(prefs.kolom)) { it.kolom }, today).today
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
                    "kolom" to s.kolom,
                    "start" to start,
                    "title" to "Ibadah dimulai ${clock(s.time)} WITA",
                    "text" to s.fullTitle + if (s.place.isNotEmpty()) " · ${s.place}" else "",
                    "detail" to listOfNotNull(
                        s.fullTitle,
                        s.place.ifEmpty { null },
                        s.leader.ifEmpty { null }?.let { "Pemimpin: $it" },
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
        val kolom = inputData.getInt("kolom", 0)
        val start = inputData.getLong("start", 0)
        // the kolom choice may have changed since the reminder was queued (the
        // same fallback as when queuing: a deleted kolom means all koloms); and
        // Android may run it late (Doze): never remind about a started service
        // (null from effectiveKolom means "all koloms", so no ?: fallback here)
        val chosen = Repository(applicationContext).cached().let { if (it == null) prefs.kolom else it.effectiveKolom(prefs.kolom) }
        val started = start > 0 && System.currentTimeMillis() >= start
        if (!prefs.notifyServices || id in prefs.remindedServices || started || (chosen != null && chosen != kolom)) {
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
