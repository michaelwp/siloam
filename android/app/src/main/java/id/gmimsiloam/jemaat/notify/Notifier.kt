package id.gmimsiloam.jemaat.notify

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import id.gmimsiloam.jemaat.MainActivity
import id.gmimsiloam.jemaat.R

/** Notification channels and posting (the user controls each channel in Android settings too). */
object Notifier {
    const val CHANNEL_SERVICES = "ibadah"
    const val CHANNEL_BIRTHDAYS = "ulang_tahun"

    const val ID_DAILY_SERVICES = 1
    const val ID_DAILY_BIRTHDAYS = 2

    /** Extra telling MainActivity which tab to open. */
    const val EXTRA_TAB = "tab"

    fun createChannels(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = context.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_SERVICES, "Jadwal ibadah", NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = "Ibadah hari ini dan pengingat sebelum ibadah dimulai" },
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_BIRTHDAYS, "Ulang tahun", NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = "Anggota jemaat yang berulang tahun hari ini" },
        )
    }

    fun canPost(context: Context): Boolean =
        (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED) &&
            NotificationManagerCompat.from(context).areNotificationsEnabled()

    /**
     * Posts a notification; [lines] become the expanded (inbox) view, so a
     * long list stays readable. Opens the app on [tab] when tapped.
     */
    fun post(context: Context, id: Int, channel: String, title: String, text: String, lines: List<String>, tab: String) {
        if (!canPost(context)) return
        val open = PendingIntent.getActivity(
            context,
            id,
            Intent(context, MainActivity::class.java)
                .putExtra(EXTRA_TAB, tab)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val style = NotificationCompat.InboxStyle().setBigContentTitle(title)
        lines.take(7).forEach { style.addLine(it) }
        if (lines.size > 7) style.setSummaryText("+${lines.size - 7} lainnya")
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
