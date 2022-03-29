// Code generated by github.com/seventv/dataloaden, DO NOT EDIT.

package loaders

import (
	"sync"
	"time"

	"github.com/SevenTV/Common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BatchEmoteLoaderConfig captures the config to create a new BatchEmoteLoader
type BatchEmoteLoaderConfig struct {
	// Fetch is a method that provides the data for the loader
	Fetch func(keys []primitive.ObjectID) ([][]*structures.Emote, []error)

	// Wait is how long wait before sending a batch
	Wait time.Duration

	// MaxBatch will limit the maximum number of keys to send in one batch, 0 = not limit
	MaxBatch int
}

// NewBatchEmoteLoader creates a new BatchEmoteLoader given a fetch, wait, and maxBatch
func NewBatchEmoteLoader(config BatchEmoteLoaderConfig) *BatchEmoteLoader {
	return &BatchEmoteLoader{
		fetch:    config.Fetch,
		wait:     config.Wait,
		maxBatch: config.MaxBatch,
	}
}

// BatchEmoteLoader batches requests
type BatchEmoteLoader struct {
	// this method provides the data for the loader
	fetch func(keys []primitive.ObjectID) ([][]*structures.Emote, []error)

	// how long to done before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// INTERNAL

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *batchEmoteLoaderBatch

	// mutex to prevent races
	mu sync.Mutex
}

type batchEmoteLoaderBatch struct {
	keys    []primitive.ObjectID
	data    [][]*structures.Emote
	error   []error
	closing bool
	done    chan struct{}
}

// Load a Emote by key, batching and caching will be applied automatically
func (l *BatchEmoteLoader) Load(key primitive.ObjectID) ([]*structures.Emote, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a Emote.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *BatchEmoteLoader) LoadThunk(key primitive.ObjectID) func() ([]*structures.Emote, error) {
	l.mu.Lock()
	if l.batch == nil {
		l.batch = &batchEmoteLoaderBatch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)
	l.mu.Unlock()

	return func() ([]*structures.Emote, error) {
		<-batch.done

		var data []*structures.Emote
		if pos < len(batch.data) {
			data = batch.data[pos]
		}

		var err error
		// its convenient to be able to return a single error for everything
		if len(batch.error) == 1 {
			err = batch.error[0]
		} else if batch.error != nil {
			err = batch.error[pos]
		}

		return data, err
	}
}

// LoadAll fetches many keys at once. It will be broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *BatchEmoteLoader) LoadAll(keys []primitive.ObjectID) ([][]*structures.Emote, []error) {
	results := make([]func() ([]*structures.Emote, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	emotes := make([][]*structures.Emote, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		emotes[i], errors[i] = thunk()
	}
	return emotes, errors
}

// LoadAllThunk returns a function that when called will block waiting for a Emotes.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *BatchEmoteLoader) LoadAllThunk(keys []primitive.ObjectID) func() ([][]*structures.Emote, []error) {
	results := make([]func() ([]*structures.Emote, error), len(keys))
	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}
	return func() ([][]*structures.Emote, []error) {
		emotes := make([][]*structures.Emote, len(keys))
		errors := make([]error, len(keys))
		for i, thunk := range results {
			emotes[i], errors[i] = thunk()
		}
		return emotes, errors
	}
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *batchEmoteLoaderBatch) keyIndex(l *BatchEmoteLoader, key primitive.ObjectID) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *batchEmoteLoaderBatch) startTimer(l *BatchEmoteLoader) {
	time.Sleep(l.wait)
	l.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		l.mu.Unlock()
		return
	}

	l.batch = nil
	l.mu.Unlock()

	b.end(l)
}

func (b *batchEmoteLoaderBatch) end(l *BatchEmoteLoader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}