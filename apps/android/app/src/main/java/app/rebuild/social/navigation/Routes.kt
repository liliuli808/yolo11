package app.rebuild.social.navigation

sealed class Routes(val route: String) {
    data object Welcome : Routes("welcome")
    data object EmailSignIn : Routes("email_signin")
    data object Verification : Routes("verification")
    data object Feed : Routes("feed")
    data object PostDetail : Routes("post_detail/{postId}") {
        const val ARG = "postId"
        fun build(postId: String) = "post_detail/$postId"
    }
    data object Persona : Routes("persona")
    data object Settings : Routes("settings")
    data object Inbox : Routes("inbox")
    data object Profile : Routes("profile")
    data object Flash : Routes("flash")
    data object ChatRoomList : Routes("chat_room_list")
    data object Chat : Routes("chat/{peerId}") {
        const val ARG = "peerId"
        fun build(peerId: String) = "chat/$peerId"
    }
}
