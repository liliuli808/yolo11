package app.rebuild.social.core.session

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

private const val SESSION_STORE_NAME = "lantern_session"

private val Context.sessionDataStore: DataStore<Preferences> by preferencesDataStore(name = SESSION_STORE_NAME)

@Singleton
class DataStoreSessionStore @Inject constructor(
    @ApplicationContext private val context: Context
) : SessionStore {

    private val dataStore = context.sessionDataStore

    private object Keys {
        val ACCESS_TOKEN = stringPreferencesKey("access_token")
        val REFRESH_TOKEN = stringPreferencesKey("refresh_token")
        val USER_ID = stringPreferencesKey("user_id")
        val ACTIVE_PERSONA_ID = stringPreferencesKey("active_persona_id")
        val EXPIRES_AT = longPreferencesKey("expires_at")
    }

    override val session: Flow<Session?> = dataStore.data.map { prefs ->
        val accessToken = prefs[Keys.ACCESS_TOKEN]
        if (accessToken.isNullOrBlank()) {
            null
        } else {
            Session(
                accessToken = accessToken,
                refreshToken = prefs[Keys.REFRESH_TOKEN],
                userId = prefs[Keys.USER_ID] ?: "",
                activePersonaId = prefs[Keys.ACTIVE_PERSONA_ID],
                expiresAt = prefs[Keys.EXPIRES_AT]?.let { Instant.ofEpochMilli(it) }
            )
        }
    }

    override suspend fun save(session: Session) {
        dataStore.edit { prefs ->
            prefs[Keys.ACCESS_TOKEN] = session.accessToken
            session.refreshToken?.let { prefs[Keys.REFRESH_TOKEN] = it } ?: prefs.remove(Keys.REFRESH_TOKEN)
            prefs[Keys.USER_ID] = session.userId
            session.activePersonaId?.let { prefs[Keys.ACTIVE_PERSONA_ID] = it } ?: prefs.remove(Keys.ACTIVE_PERSONA_ID)
            session.expiresAt?.let { prefs[Keys.EXPIRES_AT] = it.toEpochMilli() } ?: prefs.remove(Keys.EXPIRES_AT)
        }
    }

    override suspend fun clear() {
        dataStore.edit { prefs ->
            prefs.remove(Keys.ACCESS_TOKEN)
            prefs.remove(Keys.REFRESH_TOKEN)
            prefs.remove(Keys.USER_ID)
            prefs.remove(Keys.ACTIVE_PERSONA_ID)
            prefs.remove(Keys.EXPIRES_AT)
        }
    }

    override suspend fun current(): Session? = session.first()
}
