package app.rebuild.social.core.session

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

class FakeSessionStore(initial: Session? = null) : SessionStore {

    private val _session = MutableStateFlow(initial)
    override val session: Flow<Session?> = _session.asStateFlow()

    override suspend fun save(session: Session) {
        _session.value = session
    }

    override suspend fun clear() {
        _session.value = null
    }

    override suspend fun current(): Session? = _session.value
}
