package id.gmimsiloam.jemaat.notify

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.text.SpannableStringBuilder
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import id.gmimsiloam.jemaat.BuildConfig
import id.gmimsiloam.jemaat.MainActivity
import id.gmimsiloam.jemaat.R
import id.gmimsiloam.jemaat.data.NOTIFY_AHEAD_DAYS

/** Notification channels and posting (the user controls each channel in Android settings too). */
object Notifier {
    const val CHANNEL_SERVICES = "ibadah"
    const val CHANNEL_BIRTHDAYS = "ulang_tahun"
    const val CHANNEL_UPDATES = "pembaruan"

    const val ID_DAILY_SERVICES = 1
    const val ID_DAILY_BIRTHDAYS = 2
    const val ID_UPDATE = 3

    /** Extra telling MainActivity which tab to open. */
    const val EXTRA_TAB = "tab"

    fun createChannels(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = context.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_SERVICES, "Jadwal ibadah", NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = "Ibadah hari ini dan $NOTIFY_AHEAD_DAYS hari ke depan, dan pengingat sebelum ibadah dimulai" },
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_BIRTHDAYS, "Ulang tahun", NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = "Anggota jemaat yang berulang tahun hari ini dan $NOTIFY_AHEAD_DAYS hari ke depan" },
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_UPDATES, "Pembaruan aplikasi", NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = "Versi baru aplikasi tersedia untuk diunduh" },
        )
    }

    fun canPost(context: Context): Boolean =
        (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED) &&
            NotificationManagerCompat.from(context).areNotificationsEnabled()

    /** "A new version is available": tapping it opens the download link in the browser. */
    fun postUpdate(context: Context, version: String, url: String) {
        if (!canPost(context)) return
        val open = PendingIntent.getActivity(
            context,
            ID_UPDATE,
            Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val text = "Versi $version tersedia (terpasang: ${BuildConfig.VERSION_NAME}). Ketuk untuk mengunduh."
        val n = NotificationCompat.Builder(context, CHANNEL_UPDATES)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle("Versi baru Jemaat Siloam")
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setContentIntent(open)
            .setAutoCancel(true)
            .build()
        try {
            NotificationManagerCompat.from(context).notify(ID_UPDATE, n)
        } catch (_: SecurityException) {
            // permission revoked between the check and the post
        }
    }

    /**
     * Posts a notification; [lines] become the expanded view, one per line
     * (big text, so a week's list is not cut off after a few lines as in an
     * inbox view). Opens the app on [tab] when tapped.
     */
    fun post(context: Context, id: Int, channel: String, title: String, text: String, lines: List<CharSequence>, tab: String) {
        if (!canPost(context)) return
        val open = PendingIntent.getActivity(
            context,
            id,
            Intent(context, MainActivity::class.java)
                .putExtra(EXTRA_TAB, tab)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val body = SpannableStringBuilder()
        lines.forEachIndexed { i, line -> if (i > 0) body.append("\n"); body.append(line) }
        val style = NotificationCompat.BigTextStyle().setBigContentTitle(title).bigText(body)
        val n = NotificationCompat.Builder(context, channel)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(style)
            .setContentIntent(open)
            .setAutoCancel(true)
            .setCategory(NotificationCompat.CATEGORY_REMINDER)
            .build()
        try {
            NotificationManagerCompat.from(context).notify(id, n)
        } catch (_: SecurityException) {
            // permission revoked between the check and the post
        }
    }
}
