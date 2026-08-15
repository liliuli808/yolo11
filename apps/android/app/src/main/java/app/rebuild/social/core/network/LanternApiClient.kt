package app.rebuild.social.core.network

import javax.inject.Inject

class LanternApiClient @Inject constructor(
    private val service: LanternApiService
) : ApiClient {

    override suspend fun signInWithEmail(request: SignInWithEmailRequest): Result<SignInWithEmailResponse> {
        return runApiCall { service.signInWithEmail(request) }
    }

    override suspend fun verifyEmail(request: VerifyEmailRequest): Result<VerifyEmailResponse> {
        return runApiCall { service.verifyEmail(request) }
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
