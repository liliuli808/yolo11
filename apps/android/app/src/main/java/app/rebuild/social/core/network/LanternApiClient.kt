package app.rebuild.social.core.network

import java.util.UUID
import javax.inject.Inject

class LanternApiClient @Inject constructor(
    private val service: LanternApiService
) : ApiClient {

    override suspend fun register(request: RegisterRequest): Result<AuthSession> {
        return runApiCall { service.register(UUID.randomUUID().toString(), request) }
    }

    override suspend fun login(request: LoginRequest): Result<AuthSession> {
        return runApiCall { service.login(UUID.randomUUID().toString(), request) }
    }

    private suspend fun <T : Any> runApiCall(call: suspend () -> retrofit2.Response<T>): Result<T> {
        return try {
            val response = call()
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(
                        ApiException(
                            ApiError.Malformed(
                                message = "The server returned an empty response.",
                                requestId = response.headers()["X-Request-Id"]
                            )
                        )
                    )
                }
            } else {
                Result.failure(ApiException(response.toApiError()))
            }
        } catch (e: Exception) {
            Result.failure(ApiException(e.toApiError()))
        }
    }
}
