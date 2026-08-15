# Android Foundation Design

## Goal

Create a runnable Android client foundation for the clean-room social product. The
first milestone is a debug APK that launches on an emulator, uses the configured
API base URL, and can navigate between unauthenticated and authenticated shells.
No product content, original APK assets, branding, or production credentials are
included.

## Scope

- Create a single Gradle `app` module using Kotlin and Jetpack Compose.
- Use Hilt for dependency injection, Retrofit/OkHttp for HTTP, and DataStore for
  local session state.
- Provide app theme tokens, loading, empty, error, button, input, card, and avatar
  primitives.
- Provide a root navigation graph with welcome, email sign-in, verification,
  persona, feed, and settings placeholder destinations.
- Configure `API_BASE_URL` through a Gradle build configuration field, with a
  localhost-emulator default of `http://10.0.2.2:8081/`.
- Add unit and Compose navigation tests that use fake dependencies rather than a
  running API.

## Boundaries

- The module remains `apps/android/app`; packages create clear boundaries for
  `core/design`, `core/network`, `core/session`, `navigation`, and feature shells.
- The app owns UI state and maps API errors to user-safe messages. API calls are
  implemented behind interfaces so feature work can replace placeholders without
  rewriting navigation or session persistence.
- The foundation does not implement real authentication, persona CRUD, feed data,
  media uploads, or moderation actions. Those follow once their server APIs are
  available.

## Runtime Flow

1. `MainActivity` creates the Compose root and injects the session store.
2. The root restores locally persisted session state.
3. Restored sessions enter the authenticated shell; otherwise the welcome route is
   shown.
4. Navigation destinations render feature placeholders until their corresponding
   API-backed features are added.
5. Network clients read `BuildConfig.API_BASE_URL`; debug builds targeting the
   emulator reach the local API through `10.0.2.2`.

## Error Handling

- Retrofit/OkHttp requests surface normalized API errors containing code, message,
  and request ID.
- Connection failures, malformed responses, and expired sessions map to distinct
  UI states.
- A future authenticator can refresh sessions once the feature authentication
  repository is implemented; the foundation exposes the required session interface
  but does not make fake refresh calls.

## Verification

- `./gradlew :app:assembleDebug` builds the APK.
- `./gradlew :app:testDebugUnitTest` passes session and network mapping tests.
- `./gradlew :app:connectedDebugAndroidTest` verifies root navigation and the
  loading/error/empty components on an emulator.
- `./gradlew :app:installDebug` installs the app; launching it shows the welcome
  route when no session is stored.

## Local Run Prerequisites

- Android Studio with an Android SDK and an emulator (API 29 or newer).
- A JDK compatible with the selected Android Gradle Plugin.
- For local API access, start the backend with `make dev` and keep the emulator
  base URL as `http://10.0.2.2:8081/`.
