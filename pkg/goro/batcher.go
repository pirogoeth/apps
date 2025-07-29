package goro

// Batcher takes a receiving channel and reads items off of it as they are received.
// It then holds these items until `batchSize` number of items are queued, then sends
// them on a sending channel.
type Batcher[T any] struct {
	receiver  <-chan T
	queue     []T
	batchSize uint
}

func NewBatcher[T any](batchSize uint, receiver <-chan T) *Batcher[T] {
	return &Batcher[T]{
		batchSize: batchSize,
		receiver:  receiver,
		queue:     make([]T, batchSize*2),
	}
}
