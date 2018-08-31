package kamux

import (
	"log"
	"sync"

	"github.com/Shopify/sarama"
)

// KamuxWorker represents a worker for a given kafka partition
// It will process messages from the given partition sent by the parent
// SaramaConsumerProducer class
type KamuxWorker struct {
	workQueue       chan *sarama.ConsumerMessage
	messagesTreated int64
	wg              *sync.WaitGroup
	parent          *Kamux
	lastOffset      int64
}

// NewKamuxWorker creates a new workerand link it to
// it's SaramaConsumerProducer parent
func NewKamuxWorker(parentKamux *Kamux) (pw *KamuxWorker) {
	pw = new(KamuxWorker)
	pw.workQueue = make(chan *sarama.ConsumerMessage, 10000)
	pw.wg = new(sync.WaitGroup)
	pw.wg.Add(1)
	pw.parent = parentKamux

	// Auto-launch
	go pw.EventDispatcher()

	return
}

// EventDispatcher will process work from worker queue
// and exec handler on each message received
func (pw *KamuxWorker) EventDispatcher() {

	for message := range pw.workQueue {
		// Exec handler
		err := pw.parent.Config.Handler(message)
		if err != nil {
			log.Printf("[SCP       ] Error handling message : %s", err)
			pw.parent.StopWithError(err)
			break
		}

		// Increment messages treated
		pw.messagesTreated++
		pw.lastOffset = message.Offset

		// Markoffset if user wants to
		if pw.parent.Config.MarkOffsets {
			pw.parent.kafkaConsumer.MarkOffset(message, "")
		}
	}

	// Work is done
	pw.wg.Done()
	pw.parent.waitGroup.Done()

	return
}

// Enqueue will enqueue a sarama consumer message in the
// work queue of the partition worker
func (pw *KamuxWorker) Enqueue(cm *sarama.ConsumerMessage) {
	pw.workQueue <- cm
}

// MessagesProcessed returns the number of message treated by
// this worker since startup
func (pw *KamuxWorker) MessagesProcessed() int64 {
	return pw.messagesTreated
}

// Stop is a synchronous function that will stop
// the worker processing, and will wait for the remaining messages
// to be treated
func (pw *KamuxWorker) Stop() {

	// We close channel
	close(pw.workQueue)

	// We wait
	pw.wg.Wait()

	return
}