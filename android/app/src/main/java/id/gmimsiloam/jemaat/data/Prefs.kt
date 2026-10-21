package id.gmimsiloam.jemaat.data

import android.content.Context
import androidx.core.content.edit

/** The user's settings, stored on the device. */
class Prefs(context: Context) {
    private val p = context.getSharedPreferences("settings", Context.MODE_PRIVATE)

    var serverUrl: String
        get() = p.getString("serverUrl", DEFAULT_SERVER) ?: DEFAULT_SERVER
        set(v) = p.edit { putString("serverUrl", v.trim().trimEnd('/')) }

    /** The kolom shown and notified about; null = all koloms. */
    var kolom: Int?
        get() = p.getInt("kolom", 0).takeIf { it > 0 }
        set(v) = p.edit { putInt("kolom", v ?: 0) }

    var notifyServices: Boolean
        get() = p.getBoolean("notifyServices", true)
        set(v) = p.edit { putBoolean("notifyServices", v) }

    var notifyBirthdays: Boolean
        get() = p.getBoolean("notifyBirthdays", true)
        set(v) = p.edit { putBoolean("notifyBirthdays", v) }

    /** Minutes before a service for the reminder; 0 = no reminder. */
    var remindBeforeMinutes: Int
        get() = p.getInt("remindBefore", 60)
        set(v) = p.edit { putInt("remindBefore", v) }

    /** When the daily summary is shown, as minutes after midnight (default 06:00). */
    var dailyAtMinutes: Int
        get() = p.getInt("dailyAt", 6 * 60)
        set(v) = p.edit { putInt("dailyAt", v) }

    /** The dates (yyyy-MM-dd) today's services / birthdays were last notified, so each shows once a day. */
    var lastDailyServices: String
        get() = p.getString("lastDailyServices", "") ?: ""
        set(v) = p.edit { putString("lastDailyServices", v) }

    var lastDailyBirthdays: String
        get() = p.getString("lastDailyBirthdays", "") ?: ""
        set(v) = p.edit { putString("lastDailyBirthdays", v) }

    /** Whether the notification permission was already asked for (asked once; then via settings). */
    var askedNotificationPermission: Boolean
        get() = p.getBoolean("askedNotifications", false)
        set(v) = p.edit { putBoolean("askedNotifications", v) }

    /** IDs of services already reminded about (kept small: today's only). */
    var remindedServices: Set<String>
        get() = p.getStringSet("reminded", emptySet()) ?: emptySet()
        set(v) = p.edit { putStringSet("reminded", v) }

    /** Beranda's open accordion sections (both open at first). */
    var openSections: Set<String>
        get() = p.getStringSet("openSections", null) ?: setOf("notices", "stats")
        set(v) = p.edit { putStringSet("openSections", v) }

    companion object {
        const val DEFAULT_SERVER = "https://siloam.fly.dev"
    }
}
