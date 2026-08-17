package app.rebuild.social.core.network

import retrofit2.Response
import retrofit2.http.Body
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
    val username: String,
    val password: String,
    val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class LoginRequest(
    val username: String,
    val password: String,
    val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class AuthSession(
    val accessToken: String,
    val refreshToken: String,
    val tokenType: String,
    val expiresIn: Int,
    val userId: String,
    val personaId: String? = null,
    val isStaff: Boolean = false
)

/**
 * Retrofit service interface used by [ApiClient]. Feature work can add services alongside
 * this one without changing the network wiring.
 */
interface LanternApiService {
    @POST("v1/auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthSession>

    @POST("v1/auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthSession>
}
