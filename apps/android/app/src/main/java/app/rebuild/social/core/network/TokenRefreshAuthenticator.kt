package app.rebuild.social.core.network

import app.rebuild.social.core.session.SessionStore
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import javax.inject.Inject

/**
 * Placeholder for token refresh. The foundation exposes the required [SessionStore]
 * dependency and interface so a real refresh implementation can be added once the
 * authentication repository is ready. This authenticator does not make fake refresh
 * calls; it reports the failure so callers can surface a session-expired UI state.
 */
class TokenRefreshAuthenticator @Inject constructor(
    private val sessionStore: SessionStore
) : Authenticator {

    override fun authenticate(route: Route?, response: Response): Request? {
        // Foundation milestone: do not implement fake refresh. Future implementation:
        // 1. Read refresh token from sessionStore.
        // 2. Call auth repository refresh endpoint.
        // 3. Persist new tokens via sessionStore.
        // 4. Retry request with new access token.
        return null
    }
}
