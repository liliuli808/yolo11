package app.rebuild.social.feature.auth

import app.rebuild.social.core.network.ApiClient
import app.rebuild.social.core.network.AuthSession
import app.rebuild.social.core.network.LoginRequest
import app.rebuild.social.core.network.RegisterRequest
import app.rebuild.social.core.session.Session
import app.rebuild.social.core.session.SessionStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class AuthViewModelTest {

    private val dispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private val sampleSession = AuthSession(
        accessToken = "access-token",
        refreshToken = "refresh-token",
        tokenType = "Bearer",
        expiresIn = 3600,
        userId = "user-1",
        personaId = "persona-1",
        isStaff = false
    )

    @Test
    fun `login success saves session and invokes callback`() = runTest(dispatcher) {
        val api = FakeApiClient(Result.success(sampleSession))
        val store = FakeSessionStore()
        val viewModel = AuthViewModel(api, store)
        var invoked = false

        viewModel.login("alice", "password123", "token", { invoked = true })
        advanceUntilIdle()

        assertEquals(1, api.loginCalls)
        assertTrue(invoked)
        assertEquals("user-1", store.current()?.userId)
        assertEquals("access-token", store.current()?.accessToken)
        assertEquals("persona-1", store.current()?.activePersonaId)
    }

    @Test
    fun `login failure sets error state and does not save session`() = runTest(dispatcher) {
        val api = FakeApiClient(Result.failure(IllegalStateException("boom")))
        val store = FakeSessionStore()
        val viewModel = AuthViewModel(api, store)
        var invoked = false

        viewModel.login("alice", "password123", "token", { invoked = true })
        advanceUntilIdle()

        val error = viewModel.uiState.value as AuthUiState.Error
        assertEquals("boom", error.message)
        assertFalse(invoked)
        assertNull(store.current())
    }

    @Test
    fun `register sends register request with turnstile token`() = runTest(dispatcher) {
        val api = FakeApiClient(Result.success(sampleSession))
        val store = FakeSessionStore()
        val viewModel = AuthViewModel(api, store)

        viewModel.register("alice", "password123", "token", {})
        advanceUntilIdle()

        assertEquals(1, api.registerCalls)
        val request = api.lastRegisterRequest!!
        assertEquals("alice", request.username)
        assertEquals("password123", request.password)
        assertEquals("token", request.turnstileToken)
    }

    @Test
    fun `submit is ignored while loading`() = runTest(dispatcher) {
        val api = FakeApiClient(Result.success(sampleSession))
        val store = FakeSessionStore()
        val viewModel = AuthViewModel(api, store)

        viewModel.login("alice", "password123", "token", {})
        viewModel.login("bob", "password456", "token2", {})
        advanceUntilIdle()

        assertEquals(1, api.loginCalls)
    }

    @Test
    fun `clearError resets error state to idle`() = runTest(dispatcher) {
        val api = FakeApiClient(Result.failure(IllegalStateException("boom")))
        val store = FakeSessionStore()
        val viewModel = AuthViewModel(api, store)

        viewModel.login("alice", "password123", "token", {})
        advanceUntilIdle()
        assertTrue(viewModel.uiState.value is AuthUiState.Error)

        viewModel.clearError()
        assertTrue(viewModel.uiState.value is AuthUiState.Idle)
    }
}

private class FakeApiClient(private val result: Result<AuthSession>) : ApiClient {
    var loginCalls = 0
    var registerCalls = 0
    var lastRegisterRequest: RegisterRequest? = null

    override suspend fun login(request: LoginRequest): Result<AuthSession> {
        loginCalls++
        return result
    }

    override suspend fun register(request: RegisterRequest): Result<AuthSession> {
        registerCalls++
        lastRegisterRequest = request
        return result
    }
}

private class FakeSessionStore(initial: Session? = null) : SessionStore {
    private val flow = MutableStateFlow(initial)

    override val session: Flow<Session?> = flow

    override suspend fun save(session: Session) {
        flow.value = session
    }

    override suspend fun clear() {
        flow.value = null
    }

    override suspend fun current(): Session? = flow.value
}
