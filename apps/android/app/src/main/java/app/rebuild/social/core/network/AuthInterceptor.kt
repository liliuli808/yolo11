package app.rebuild.social.core.network

import app.rebuild.social.core.session.SessionStore
import kotlinx.coroutines.runBlocking
import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject

/**
 * Attaches the current access token to outgoing requests. Runs synchronously because
 * OkHttp interceptors are blocking; [SessionStore.current] is a short-lived DataStore
 * read.
 */
class AuthInterceptor @Inject constructor(
    private val sessionStore: SessionStore
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val session = runBlocking { sessionStore.current() }

        return if (session?.accessToken != null) {
            val authenticated = request.newBuilder()
                .header("Authorization", "Bearer ${session.accessToken}")
                .build()
            chain.proceed(authenticated)
        } else {
            chain.proceed(request)
        }
    }
}
