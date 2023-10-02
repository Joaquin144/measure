package sh.measure.android.exceptions

import sh.measure.android.MeasureClient
import sh.measure.android.logger.LogLevel
import sh.measure.android.logger.Logger
import sh.measure.android.utils.TimeProvider
import sh.measure.android.events.EventTracker
import java.lang.Thread.UncaughtExceptionHandler

/**
 * An [UncaughtExceptionHandler] that tracks unhandled exceptions and reports them
 * to [MeasureClient].
 *
 * In case an exception is raised during parsing of the uncaught exception, tracking silently fails
 * to avoid an infinite loop of exceptions.
 *
 * Note that the original [UncaughtExceptionHandler] is always called.
 */
internal class UnhandledExceptionCollector(
    private val logger: Logger,
    private val eventTracker: EventTracker,
    private val timeProvider: TimeProvider,
) : UncaughtExceptionHandler {

    private val originalHandler: UncaughtExceptionHandler? =
        Thread.getDefaultUncaughtExceptionHandler()

    /**
     * Registers [UnhandledExceptionCollector] as the [UncaughtExceptionHandler].
     */
    fun register() {
        logger.log(LogLevel.Debug, "Registering exception handler")
        Thread.setDefaultUncaughtExceptionHandler(this)
    }

    override fun uncaughtException(thread: Thread, throwable: Throwable) {
        logger.log(LogLevel.Debug, "Unhandled exception received")
        try {
            val measureException = ExceptionFactory.createMeasureException(
                throwable,
                handled = false,
                timestamp = timeProvider.currentTimeSinceEpochInMillis,
                thread = thread,
            )
            eventTracker.trackUnhandledException(measureException)
        } catch (e: Throwable) {
            // Prevent an infinite loop of exceptions if the above code fails.
            logger.log(LogLevel.Error, "Failed to track exception", e)
        } finally {
            // Call the original handler so that we do not swallow any exceptions.
             originalHandler?.uncaughtException(thread, throwable)
        }
    }
}

