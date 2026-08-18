package app.rebuild.social.core.network

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNamingStrategy
import org.junit.Assert.assertEquals
import org.junit.Test

class RegisterRequestTest {

    @OptIn(ExperimentalSerializationApi::class)
    private val json = Json { namingStrategy = JsonNamingStrategy.SnakeCase }

    @OptIn(ExperimentalSerializationApi::class)
    @Test
    fun `register wire format locks inviteCode key as invite_code`() {
        val request = RegisterRequest("Alice_1", "password123", "tok", "LANTERN-ABCD")
        val encoded = json.encodeToString<RegisterRequest>(request)
        assertEquals(1, Regex("\"invite_code\"\\s*:").findAll(encoded).count())
        assertEquals(0, Regex("\"inviteCode\"\\s*:").findAll(encoded).count())
    }
}