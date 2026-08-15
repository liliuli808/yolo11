package app.rebuild.social.core.network

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

/**
 * Low-level API client. Feature modules will add their own service interfaces; this
 * interface only exposes the endpoints required by the foundation (auth handshake).
 */
interface ApiClient {
    suspend fun signInWithEmail(request: SignInWithEmailRequest): Result<SignInWithEmailResponse>
    suspend fun verifyEmail(request: VerifyEmailRequest): Result<VerifyEmailResponse>
}

@kotlinx.serialization.Serializable
data class SignInWithEmailRequest(
    val email: String
)

@kotlinx.serialization.Serializable
data class SignInWithEmailResponse(
    val email: String,
    val expiresInSeconds: Int
)

@kotlinx.serialization.Serializable
data class VerifyEmailRequest(
    val email: String,
    val code: String
)

@kotlinx.serialization.Serializable
data class VerifyEmailResponse(
    val accessToken: String,
    val refreshToken: String? = null,
    val userId: String,
    val activePersonaId: String? = null,
    val expiresAt: String? = null
)

/**
 * Retrofit service interface used by [ApiClient]. Feature work can add services alongside
 * this one without changing the network wiring.
 */
interface LanternApiService {
    @POST("v1/auth/email/signin")
    suspend fun signInWithEmail(@Body request: SignInWithEmailRequest): Response<SignInWithEmailResponse>

    @POST("v1/auth/email/verify")
    suspend fun verifyEmail(@Body request: VerifyEmailRequest): Response<VerifyEmailResponse>
}
