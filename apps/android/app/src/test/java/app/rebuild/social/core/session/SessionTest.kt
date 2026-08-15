package app.rebuild.social.core.session

import app.cash.turbine.test
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

class SessionTest {

    @Test
    fun `session is expired when expiresAt is in the past`() {
        val session = Session(
            accessToken = "token",
            refreshToken = null,
            userId = "user-1",
            activePersonaId = null,
            expiresAt = Instant.now().minusSeconds(60)
        )
        assertTrue(session.isExpired)
    }

    @Test
    fun `session is not expired when expiresAt is in the future`() {
        val session = Session(
            accessToken = "token",
            refreshToken = null,
            userId = "user-1",
            activePersonaId = null,
            expiresAt = Instant.now().plusSeconds(60)
        )
        assertFalse(session.isExpired)
    }

    @Test
    fun `session without expiresAt is not expired`() {
        val session = Session(
            accessToken = "token",
            refreshToken = null,
            userId = "user-1",
            activePersonaId = null,
            expiresAt = null
        )
        assertFalse(session.isExpired)
    }
}

class FakeSessionStoreTest {

    @Test
    fun `store emits null by default`() = runTest {
        val store = FakeSessionStore()
        store.session.test {
            assertNull(awaitItem())
        }
    }

    @Test
    fun `store emits saved session`() = runTest {
        val store = FakeSessionStore()
        val session = sampleSession()

        store.session.test {
            assertNull(awaitItem())
            store.save(session)
            assertEquals(session, awaitItem())
        }
    }

    @Test
    fun `clear resets session to null`() = runTest {
        val session = sampleSession()
        val store = FakeSessionStore(session)

        store.session.test {
            assertEquals(session, awaitItem())
            store.clear()
            assertNull(awaitItem())
        }
    }

    @Test
    fun `current returns latest value`() = runTest {
        val store = FakeSessionStore()
        val session = sampleSession()
        store.save(session)
        assertEquals(session, store.current())
    }
}

fun sampleSession(
    accessToken: String = "access-token",
    userId: String = "user-1"
) = Session(
    accessToken = accessToken,
    refreshToken = "refresh-token",
    userId = userId,
    activePersonaId = "persona-1",
    expiresAt = Instant.now().plusSeconds(3600)
)
