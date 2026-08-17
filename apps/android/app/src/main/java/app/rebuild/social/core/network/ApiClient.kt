package app.rebuild.social.core.network

import kotlinx.serialization.SerialName
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.Header
import retrofit2.http.POST

/**
 * Low-level API client. Feature modules will add their own service interfaces; this
 * interface only exposes the endpoints required by the foundation (auth handshake).
 */
interface ApiClient {
    suspend fun register(request: RegisterRequest): Result<AuthSession>
    suspend fun login(request: LoginRequest): Result<AuthSession>
}

@kotlinx.serialization.Serializable
data class RegisterRequest(
    @SerialName("username") val username: String,
    @SerialName("password") val password: String,
    @SerialName("turnstileToken") val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class LoginRequest(
    @SerialName("username") val username: String,
    @SerialName("password") val password: String,
    @SerialName("turnstileToken") val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class AuthSession(
    @SerialName("accessToken") val accessToken: String,
    @SerialName("refreshToken") val refreshToken: String,
    @SerialName("tokenType") val tokenType: String,
    @SerialName("expiresIn") val expiresIn: Int,
    @SerialName("userId") val userId: String,
    @SerialName("personaId") val personaId: String? = null,
    @SerialName("isStaff") val isStaff: Boolean = false
)

/**
 * Retrofit service interface used by [ApiClient]. Feature work can add services alongside
 * this one without changing the network wiring.
 */
interface LanternApiService {
    @POST("v1/auth/register")
    suspend fun register(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: RegisterRequest
    ): Response<AuthSession>

    @POST("v1/auth/login")
    suspend fun login(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: LoginRequest
    ): Response<AuthSession>
}
