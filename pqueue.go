// This package provides a priority queue implementation and
// scaffold interfaces.
//
// Addition to original package, this package adds method and
// other internals for inserting only unique items in queue
//
// Copyright (C) 2015 by Milos Mileusnic <milos@groowe.com>
//
// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
package pqueue

import (
	"container/heap"
	"errors"
	"sync"
)

// Only items implementing this interface can be enqueued
// on the priority queue.
type QueueItem interface {
	Less(other interface{}) bool
	Id() interface{}
}

// Queue is a threadsafe priority queue exchange. Here's
// a trivial example of usage:
//
//     q := pqueue.New(0)
//     go func() {
//         for {
//             task := q.Dequeue()
//             println(task.(*CustomTask).Name)
//         }
//     }()
//     for i := 0; i < 100; i := 1 {
//         task := CustomTask{Name: "foo", priority: rand.Intn(10)}
//         q.Enqueue(&task)
//     }
//
type Queue struct {
	Limit   int
	history map[interface{}]struct{}
	items   *sorter
	cond    *sync.Cond
}

// New creates and initializes a new priority queue, taking
// a limit as a parameter. If 0 given, then queue will be
// unlimited.
func New(max int) (q *Queue) {
	var locker sync.Mutex
	q = &Queue{Limit: max}
	q.history = make(map[interface{}]struct{}, 0)
	q.items = new(sorter)
	q.cond = sync.NewCond(&locker)
	heap.Init(q.items)
	return
}

// Enqueue puts given item to the queue.
// Lock the queue and calls enqueue()
func (q *Queue) Enqueue(item QueueItem) (err error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.enqueue(item)
}

// Enqueue puts given item to the queue.
func (q *Queue) enqueue(item QueueItem) (err error) {
	if q.Limit > 0 && q.Len() >= q.Limit {
		return errors.New("Queue limit reached")
	}
	q.history[item.Id()] = struct{}{}
	heap.Push(q.items, item)
	q.cond.Signal()
	return
}

// check if item already exists in queue (or it has been into queue)
func (q *Queue) ItemExists(item QueueItem) bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.idExists(item.Id())
}

func (q *Queue) IdExists(id interface{}) bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.idExists(id)
}

func (q *Queue) idExists(id interface{}) bool {
	if _, ok := q.history[id]; ok {
		return true
	} else {
		return false
	}
}

// Enqueue puts item in queue only if it hasn't already been in queue
func (q *Queue) EnqueueUnique(item QueueItem) (added bool, err error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if !q.idExists(item.Id()) {
		err = q.enqueue(item)
		added = true
	}
	return
}

/*
	Clear queue history so the elements can be EnqueueUnique again
*/
func (q *Queue) ClearHistory() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.history = nil
	q.history = make(map[interface{}]struct{}, 0)
}

func (q *Queue) RemoveFromHistory(element interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	delete(q.history, element)
}

// Dequeue takes an item from the queue. If queue is empty
// then should block waiting for at least one item.
func (q *Queue) Dequeue() (item QueueItem) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	var x interface{}
	for {
		x = heap.Pop(q.items)
		if x == nil {
			q.cond.Wait()
		} else {
			break
		}
	}
	item = x.(QueueItem)
	return
}

// Safely changes enqueued items limit. When limit is set
// to 0, then queue is unlimited.
func (q *Queue) ChangeLimit(newLimit int) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.Limit = newLimit
}

// Len returns number of enqueued elemnents.
func (q *Queue) Len() int {
	return q.items.Len()
}

// IsEmpty returns true if queue is empty.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

type sorter []QueueItem

func (s *sorter) Push(i interface{}) {
	item, ok := i.(QueueItem)
	if !ok {
		return
	}
	*s = append((*s)[:], item)
}

func (s *sorter) Pop() (x interface{}) {
	if s.Len() > 0 {
		l := s.Len() - 1
		x = (*s)[l]
		(*s)[l] = nil
		*s = (*s)[:l]
	}
	return
}

func (s *sorter) Len() int {
	return len((*s)[:])
}

func (s *sorter) Less(i, j int) bool {
	return (*s)[i].Less((*s)[j])
}

func (s *sorter) Swap(i, j int) {
	if s.Len() > 0 {
		(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
	}
}
