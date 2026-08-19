package app.rebuild.social.core.network

import kotlinx.serialization.encodeToString
import org.junit.Assert.assertEquals
import org.junit.Test

class RegisterRequestTest {

    private val json = NetworkModule.provideJson()

    @Test
    fun `register wire format sends inviteCode using API contract name`() {
        val request = RegisterRequest("Alice_1", "password123", "tok", "LANTERN-ABCD")
        val encoded = json.encodeToString<RegisterRequest>(request)
        assertEquals(1, Regex("\"inviteCode\"\\s*:").findAll(encoded).count())
        assertEquals(0, Regex("\"invite_code\"\\s*:").findAll(encoded).count())
    }
}
