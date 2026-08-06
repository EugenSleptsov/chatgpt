package app

import (
	"log"
	"sync"
)

// mailboxBuffer is how many jobs one chat may have waiting. It only fills when
// that chat's goroutine is stuck inside a long API call; once full, further
// jobs for THAT chat are dropped and every other chat keeps running.
const mailboxBuffer = 64

// dispatcher routes jobs to one mailbox goroutine per chat.
//
// It replaces the previous fixed pool of N workers with chat IDs hashed onto
// them. That pool preserved the same invariant (all mutations of a Chat happen
// on one goroutine, so no mutexes are needed) but coupled unrelated chats in
// two ways, and one slow request took the whole bot down through both:
//
//   - chats sharing a partition queued behind each other, so a chat stuck in a
//     multi-minute completion stalled every chat that hashed to its worker;
//   - the dispatch loop pushed into a bounded channel with a blocking send, so
//     once one partition backed up, the main loop stopped reading updates and
//     the bot went silent everywhere — not just in the affected chat.
//
// Here each chat owns its mailbox and its goroutine, and sends never block, so
// a hung request is contained to the one chat that made it.
//
// A chat's goroutine then lives for the life of the process. That is not a leak
// waiting to happen: the reminder scheduler touches every chat in storage every
// 30 seconds anyway, so the goroutine count tracks a set the bot already holds
// in memory, chat object for chat object.
type dispatcher struct {
	newWorker func() *Worker

	mu     sync.Mutex // guards boxes, closed, and every send into a mailbox
	boxes  map[int64]chan Job
	closed bool

	wg sync.WaitGroup
}

func newDispatcher(newWorker func() *Worker) *dispatcher {
	return &dispatcher{
		newWorker: newWorker,
		boxes:     make(map[int64]chan Job),
	}
}

// Send hands a job to the chat's mailbox, starting one on first use. It never
// blocks: a full mailbox means that chat is still busy, and the job is dropped
// rather than backing up into the caller. Reports whether the job was accepted.
//
// The send happens under the mutex — it cannot block, thanks to the default
// case — so it can never race with Close closing the same channel.
func (d *dispatcher) Send(chatID int64, job Job) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return false
	}

	box, ok := d.boxes[chatID]
	if !ok {
		box = make(chan Job, mailboxBuffer)
		d.boxes[chatID] = box
		w := d.newWorker()
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			w.Start(box)
		}()
	}

	select {
	case box <- job:
		return true
	default:
		return false
	}
}

// Close stops every mailbox. Wait blocks until in-flight jobs finish.
func (d *dispatcher) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}
	d.closed = true
	for _, box := range d.boxes {
		close(box)
	}
}

// Wait blocks until every mailbox goroutine has finished its current job.
func (d *dispatcher) Wait() { d.wg.Wait() }

// logDrop reports a dropped job. A chat only reaches this state when its
// goroutine has been stuck for a long time, so it is worth a line in the log.
func logDrop(chatID int64) {
	log.Printf("[Dispatcher] чат %d занят, очередь переполнена — задание отброшено", chatID)
}
