package app.rebuild.social.core.network

import app.rebuild.social.BuildConfig
import dagger.Binds
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class NetworkModule {

    @Binds
    @Singleton
    abstract fun bindApiClient(lanternApiClient: LanternApiClient): ApiClient

    @Binds
    @Singleton
    abstract fun bindContentApiClient(lanternContentApiClient: LanternContentApiClient): ContentApiClient

    companion object {

        @Provides
        @Singleton
        fun provideJson(): Json = Json {
            ignoreUnknownKeys = true
            coerceInputValues = true
        }

        @Provides
        @Singleton
        fun provideOkHttpClient(
            authInterceptor: AuthInterceptor,
            tokenRefreshAuthenticator: TokenRefreshAuthenticator
        ): OkHttpClient {
            return OkHttpClient.Builder()
                .addInterceptor(authInterceptor)
                .authenticator(tokenRefreshAuthenticator)
                .apply {
                    if (BuildConfig.DEBUG) {
                        addInterceptor(
                            HttpLoggingInterceptor().apply {
                                level = HttpLoggingInterceptor.Level.BASIC
                            }
                        )
                    }
                }
                .connectTimeout(30, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .writeTimeout(30, TimeUnit.SECONDS)
                .build()
        }

        @Provides
        @Singleton
        fun provideRetrofit(
            okHttpClient: OkHttpClient,
            json: Json
        ): Retrofit {
            val contentType = "application/json".toMediaType()
            return Retrofit.Builder()
                .baseUrl(BuildConfig.API_BASE_URL)
                .client(okHttpClient)
                .addConverterFactory(json.asConverterFactory(contentType))
                .build()
        }

        @Provides
        @Singleton
        fun provideLanternApiService(retrofit: Retrofit): LanternApiService = retrofit.create(LanternApiService::class.java)

        @Provides
        @Singleton
        fun provideContentApiService(retrofit: Retrofit): ContentApiService = retrofit.create(ContentApiService::class.java)
    }
}
