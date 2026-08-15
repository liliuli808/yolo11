package app.rebuild.social.core.network

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.ResponseBody
import retrofit2.HttpException
import retrofit2.Response
import java.io.IOException

private val errorJson = Json {
    ignoreUnknownKeys = true
    coerceInputValues = true
}

/**
 * Maps a [Throwable] produced by the network stack into a normalized [ApiError].
 */
fun Throwable.toApiError(requestId: String? = null): ApiError = when (this) {
    is SessionExpiredException -> ApiError.SessionExpired(requestId = requestId)

    is IOException -> ApiError.Network(
        message = localizedMessage ?: "Unable to reach the server. Please check your connection.",
        requestId = requestId
    )

    is HttpException -> {
        val response = response()
        val errorBody = response?.errorBody()
        val statusCode = response?.code() ?: code()
        parseHttpError(errorBody, statusCode, requestId ?: response?.headers()?.get("X-Request-Id"))
    }

    else -> ApiError.Unknown(
        message = localizedMessage ?: "Something went wrong. Please try again.",
        requestId = requestId
    )
}

/**
 * Maps a Retrofit [Response] failure body into an [ApiError].
 */
fun <T> Response<T>.toApiError(): ApiError {
    val requestId = headers()["X-Request-Id"]
    return parseHttpError(errorBody(), code(), requestId)
}

private fun parseHttpError(
    errorBody: ResponseBody?,
    statusCode: Int,
    requestId: String?
): ApiError {
    if (statusCode == HTTP_UNAUTHORIZED || statusCode == HTTP_FORBIDDEN) {
        return ApiError.SessionExpired(requestId = requestId)
    }

    val bodyString = try {
        errorBody?.string()
    } catch (_: IOException) {
        null
    }

    if (bodyString.isNullOrBlank()) {
        return ApiError.Http(
            code = "http_error",
            message = "The server returned an unexpected response (HTTP $statusCode).",
            statusCode = statusCode,
            requestId = requestId
        )
    }

    val parsed = try {
        errorJson.decodeFromString(ErrorResponse.serializer(), bodyString)
    } catch (_: SerializationException) {
        null
    }

    return if (parsed != null) {
        if (parsed.code == ERROR_CODE_SESSION_EXPIRED || parsed.code == ERROR_CODE_UNAUTHORIZED) {
            ApiError.SessionExpired(requestId = parsed.requestId ?: requestId)
        } else {
            ApiError.Http(
                code = parsed.code,
                message = parsed.message,
                statusCode = statusCode,
                requestId = parsed.requestId ?: requestId
            )
        }
    } else {
        ApiError.Malformed(
            message = "The server response could not be understood.",
            requestId = requestId
        )
    }
}

class SessionExpiredException : IOException("Session expired")
