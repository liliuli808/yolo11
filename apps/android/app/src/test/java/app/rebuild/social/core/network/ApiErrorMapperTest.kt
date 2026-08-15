package app.rebuild.social.core.network

import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import retrofit2.HttpException
import retrofit2.Response
import java.io.IOException

class ApiErrorMapperTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `IOException maps to network error`() {
        val cause = IOException("No connection")
        val error = cause.toApiError()
        assertTrue(error is ApiError.Network)
        assertEquals("network_error", error.code)
    }

    @Test
    fun `HttpException 401 maps to session expired`() {
        val response = Response.error<String>(401, "".toResponseBody("application/json".toMediaType()))
        val cause = HttpException(response)
        val error = cause.toApiError()
        assertTrue(error is ApiError.SessionExpired)
        assertEquals("session_expired", error.code)
    }

    @Test
    fun `HttpException with error body maps to http error`() {
        val body = json.encodeToString(
            ErrorResponse.serializer(),
            ErrorResponse(code = "invalid_email", message = "Email is not valid", requestId = "req-1")
        )
        val response = Response.error<String>(
            422,
            body.toResponseBody("application/json".toMediaType())
        )
        val error = response.toApiError()
        assertTrue(error is ApiError.Http)
        assertEquals("invalid_email", error.code)
        assertEquals("Email is not valid", error.message)
        assertEquals("req-1", error.requestId)
    }

    @Test
    fun `error body with session expired code maps to session expired`() {
        val body = json.encodeToString(
            ErrorResponse.serializer(),
            ErrorResponse(code = "session_expired", message = "Expired", requestId = "req-2")
        )
        val response = Response.error<String>(
            400,
            body.toResponseBody("application/json".toMediaType())
        )
        val error = response.toApiError()
        assertTrue(error is ApiError.SessionExpired)
        assertEquals("req-2", error.requestId)
    }

    @Test
    fun `malformed error body maps to malformed error`() {
        val response = Response.error<String>(
            500,
            "not-json".toResponseBody("application/json".toMediaType())
        )
        val error = response.toApiError()
        assertTrue(error is ApiError.Malformed)
        assertEquals("malformed_response", error.code)
    }

    @Test
    fun `user safe message returns message`() {
        val error = ApiError.Network("No connection", "req-3")
        assertEquals("No connection", error.userSafeMessage())
    }
}
