package app.rebuild.social.core.session

import kotlinx.coroutines.flow.Flow
import java.time.Instant

/**
 * A persisted user session. The access token is required; refresh token and persona
 * identifiers may be null when the corresponding features are not yet active.
 */
data class Session(
    val accessToken: String,
    val refreshToken: String?,
    val userId: String,
    val activePersonaId: String?,
    val expiresAt: Instant?
) {
    val isExpired: Boolean
        get() = expiresAt?.let { Instant.now().isAfter(it) } ?: false
}

/**
 * Local source of truth for the current session. Implementations are expected to be
 * coroutine-safe and to expose a [Flow] that emits the current session (or null).
 */
interface SessionStore {
    /**
     * Emits the current session, or null when no session is stored. Collectors receive
     * the stored value immediately.
     */
    val session: Flow<Session?>

    /**
     * Persists [session] and replaces any previously stored session.
     */
    suspend fun save(session: Session)

    /**
     * Clears any stored session. Subsequent [session] emissions will emit null until a
     * new session is saved.
     */
    suspend fun clear()

    /**
     * Returns the current session without observing changes, or null if none exists.
     */
    suspend fun current(): Session?
}
