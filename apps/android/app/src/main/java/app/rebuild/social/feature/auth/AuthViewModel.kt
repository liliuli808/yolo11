package app.rebuild.social.feature.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rebuild.social.core.network.ApiClient
import app.rebuild.social.core.network.LoginRequest
import app.rebuild.social.core.network.RegisterRequest
import app.rebuild.social.core.session.Session
import app.rebuild.social.core.session.SessionStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.temporal.ChronoUnit
import javax.inject.Inject

sealed interface AuthUiState {
    data object Idle : AuthUiState
    data object Loading : AuthUiState
    data class Error(val message: String) : AuthUiState
}

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore
) : ViewModel() {

    private val _uiState = MutableStateFlow<AuthUiState>(AuthUiState.Idle)
    val uiState: StateFlow<AuthUiState> = _uiState

    fun register(
        username: String,
        password: String,
        turnstileToken: String,
        inviteCode: String,
        onSuccess: () -> Unit
    ) {
        submit(
            { apiClient.register(RegisterRequest(username, password, turnstileToken, inviteCode)) },
            onSuccess
        )
    }

    fun login(username: String, password: String, turnstileToken: String, onSuccess: () -> Unit) {
        submit({ apiClient.login(LoginRequest(username, password, turnstileToken)) }, onSuccess)
    }

    fun clearError() {
        if (_uiState.value is AuthUiState.Error) {
            _uiState.value = AuthUiState.Idle
        }
    }

    private fun submit(block: suspend () -> Result<app.rebuild.social.core.network.AuthSession>, onSuccess: () -> Unit) {
        if (_uiState.value is AuthUiState.Loading) return
        _uiState.value = AuthUiState.Loading
        viewModelScope.launch {
            block().onSuccess { session ->
                sessionStore.save(
                    Session(
                        accessToken = session.accessToken,
                        refreshToken = session.refreshToken,
                        userId = session.userId,
                        activePersonaId = session.personaId,
                        expiresAt = Instant.now().plus(session.expiresIn.toLong(), ChronoUnit.SECONDS)
                    )
                )
                _uiState.value = AuthUiState.Idle
                onSuccess()
            }.onFailure { e ->
                _uiState.value = AuthUiState.Error(e.message ?: "Authentication failed")
            }
        }
    }
}
