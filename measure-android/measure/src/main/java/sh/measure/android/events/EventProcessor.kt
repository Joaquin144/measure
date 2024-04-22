package sh.measure.android.events

import sh.measure.android.attributes.Attribute
import sh.measure.android.attributes.AttributeProcessor
import sh.measure.android.attributes.appendAttributes
import sh.measure.android.executors.MeasureExecutorService
import sh.measure.android.exporter.EventExporter
import sh.measure.android.logger.LogLevel
import sh.measure.android.logger.Logger
import sh.measure.android.storage.EventStore
import sh.measure.android.utils.IdProvider
import sh.measure.android.utils.SessionIdProvider

/**
 * An interface for processing events. It is responsible for tracking events, processing them
 * by applying various attributes and transformations, and then eventually storing them or sending
 * them to the server.
 */
internal interface EventProcessor {
    /**
     * Tracks an event with the given data, timestamp and type.
     *
     * @param data The data to be tracked.
     * @param timestamp The timestamp of the event in milliseconds since epoch.
     * @param type The type of the event.
     */
    fun <T> track(
        data: T,
        timestamp: Long,
        type: String,
    )

    /**
     * Tracks an event with the given data, timestamp, type, attributes and attachments.
     *
     * @param data The data to be tracked.
     * @param timestamp The timestamp of the event in milliseconds since epoch.
     * @param type The type of the event.
     * @param attributes The attributes to be attached to the event.
     * @param attachments The attachments to be attached to the event.
     */
    fun <T> track(
        data: T,
        timestamp: Long,
        type: String,
        attributes: MutableMap<String, Any?> = mutableMapOf(),
        attachments: List<Attachment>? = null,
    )
}

internal class EventProcessorImpl(
    private val logger: Logger,
    private val executorService: MeasureExecutorService,
    private val eventStore: EventStore,
    private val idProvider: IdProvider,
    private val sessionIdProvider: SessionIdProvider,
    private val attributeProcessors: List<AttributeProcessor>,
    private val eventExporter: EventExporter,
) : EventProcessor {

    override fun <T> track(
        data: T,
        timestamp: Long,
        type: String,
    ) {
        track(data, timestamp, type, mutableMapOf(), null)
    }

    override fun <T> track(
        data: T,
        timestamp: Long,
        type: String,
        attributes: MutableMap<String, Any?>,
        attachments: List<Attachment>?,
    ) {
        val threadName = Thread.currentThread().name

        fun createEvent(): Event<T> {
            val id = idProvider.createId()
            val sessionId = sessionIdProvider.sessionId
            return Event(id, sessionId, timestamp, type, data, attachments, attributes)
        }

        fun applyAttributes(event: Event<T>) {
            event.appendAttribute(Attribute.THREAD_NAME, threadName)
            event.appendAttributes(attributeProcessors)
        }

        when (type) {
            EventType.ANR, EventType.EXCEPTION -> {
                val event = createEvent()
                applyAttributes(event)
                eventStore.store(event)
                eventExporter.export(event)
                logger.log(LogLevel.Debug, "Event processed: ${event.type}")
            }

            else -> {
                executorService.submit {
                    val event = createEvent()
                    applyAttributes(event)
                    eventStore.store(event)
                    logger.log(LogLevel.Debug, "Event processed: ${event.type}")
                }
            }
        }
    }
}
