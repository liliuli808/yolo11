package app.rebuild.social.feature.feed

import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

private val ISO_FORMATS = listOf(
    SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", Locale.US),
    SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US),
    SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSSXXX", Locale.US),
    SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ssXXX", Locale.US)
).onEach { it.timeZone = TimeZone.getTimeZone("UTC") }

/**
 * Renders an ISO-8601 timestamp as a Chinese relative time, e.g. 刚刚 / 3分钟前 / 2小时前 / 5天前.
 */
fun formatRelativeTime(iso: String?): String {
    if (iso.isNullOrBlank()) return ""
    val date = ISO_FORMATS.firstNotNullOfOrNull { fmt ->
        runCatching { fmt.parse(iso) }.getOrNull()
    } ?: return ""
    val diffMillis = Date().time - date.time
    val minutes = diffMillis / 60_000
    return when {
        minutes < 1 -> "刚刚"
        minutes < 60 -> "${minutes}分钟前"
        minutes < 60 * 24 -> "${minutes / 60}小时前"
        minutes < 60 * 24 * 30 -> "${minutes / (60 * 24)}天前"
        else -> SimpleDateFormat("yyyy-MM-dd", Locale.US).format(date)
    }
}
