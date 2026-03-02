package job

import (
	"errors"
	"sync"
)

var (
	ErrJobAlreadyExists = errors.New("job already in store")
	ErrJobNull          = errors.New("job is null")
	ErrJobNotFound      = errors.New("job not found")
)

// The JobId will be the key ta the Job itself
// This is a opaque object to the outside
// Note: The job should only mutate in terms
// of its timestamp or State
// TODO : replace map with our data struthure
type JobStore struct {
	buf map[ID]*Job
	mu  sync.RWMutex
}

func New() *JobStore {
	return &JobStore{
		make(map[ID]*Job),
		sync.RWMutex{},
	}
}

func (store *JobStore) Get(id ID) (*Job, error) {
	// Note: ensure we dont fire another func
	// which tries acuiring lock, that may cause a deadlock here
	store.mu.RLock()
	defer store.mu.RUnlock()

	j, ok := store.buf[id]

	if !ok {
		return nil, ErrJobNotFound
	}

	// Do a deep copy to avoid further mutations
	copy := *j

	return &copy, nil
}

// Adds a job to the JobStore
func (store *JobStore) Push(job *Job) error {
	// Now all writers and readers will block
	// until we finish mutation on the store
	store.mu.Lock()
	defer store.mu.Unlock()

	if nil == job {
		return ErrJobNull
	}

	_, ok := store.buf[job.Id]

	// If job is already with same id ,
	// sonethibg was messed up real bad ,
	// and we should prolly crash this one
	if ok {
		return ErrJobAlreadyExists
	}

	store.buf[job.Id] = job

	return nil
}

// Removes a job from the job store and returns a ref
func (store *JobStore) Pop(id ID) (*Job, error) {
	// Remarks: Same as store.Get()
	store.mu.Lock()
	defer store.mu.Unlock()

	j, ok := store.buf[id]

	// If job wasnt in the store
	if !ok {
		return nil, ErrJobNotFound
	}

	// This marks the end of lifecycle for the job itself
	// so there is no need to a deep copy
	delete(store.buf, id)

	return j, nil
}
