package app.rebuild.social.navigation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rebuild.social.core.session.Session
import app.rebuild.social.core.session.SessionStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import javax.inject.Inject

@HiltViewModel
class SessionViewModel @Inject constructor(
    sessionStore: SessionStore
) : ViewModel() {

    val session: StateFlow<Session?> = sessionStore.session
        .stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(stopTimeoutMillis = 5_000),
            initialValue = null
        )

    val isLoading: StateFlow<Boolean> = sessionStore.session
        .map { false }
        .stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(stopTimeoutMillis = 5_000),
            initialValue = true
        )
}
