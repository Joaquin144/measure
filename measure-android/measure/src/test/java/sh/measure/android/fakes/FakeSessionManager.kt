package sh.measure.android.fakes

import sh.measure.android.SessionManager

internal class FakeSessionManager : SessionManager {
    var sessionPids = mutableListOf(Pair("fake-session-id", 1234))

    override fun getSessionId(): String {
        return "fake-session-id"
    }

    override fun getSessionsForPids(): Map<Int, List<String>> {
        return sessionPids.groupBy({ it.second }, { it.first })
    }

    override fun onAppBackground() {
        // No-op
    }

    override fun onAppForeground() {
        // No-op
    }

    override fun clearOldSessions() {
        // No-op
    }
}
