package scheduler

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

type WorkPool interface {
	PushBottom(node *ImageTask)
	PopBottom() *ImageTask
	PopTop() *ImageTask
	Steal(thiefID int, workPools []WorkPool) bool
}
type node struct {
	task *ImageTask
	next *node // Pointer to the next node
	prev *node // Pointer to the previous node
}

type lfdeque struct {
	head *node // Stores *node
	tail *node // Stores *node
}

func NewNode(task *ImageTask, next *node) *node {
	n := &node{task: task}
	n.next = nil
	return n
}

func NewQueue() *lfdeque {
	sentinel := &node{} // A dummy node as sentinel
	return &lfdeque{head: sentinel, tail: sentinel}
}

func (q *lfdeque) PushBottom(task *ImageTask) {
	var expectTail, expectTailNext *node
	newTask := NewNode(task, nil)

	success := false
	for !success {

		expectTail = q.tail
		expectTailNext = expectTail.next

		// If not at the tail then try again
		if q.tail != expectTail {
			continue
		}

		// If expected tail is not nil help it along and try again
		if expectTailNext != nil {
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(expectTailNext))
			continue
		}

		// Logical enqueue
		success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail.next)), unsafe.Pointer(expectTailNext), unsafe.Pointer(newTask))

	}

	// Physical enqueue
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(newTask))
}

// Dequeue removes a ImageTask from the queue
func (q *lfdeque) PopBottom() *ImageTask {
	var dequeued *ImageTask
	var expectSentinel, expectRemoved, expectTail *node

	success := false
	for !success {
		expectSentinel = q.head
		expectRemoved = expectSentinel.next
		expectTail = q.tail

		// If not at the head then try again
		if q.head != expectSentinel {
			continue
		}

		// Signal that queue is empty when the sentinel node is reached
		if expectRemoved == nil {
			return nil
		}

		// Help tail along if it is behind and try again
		if expectTail == expectSentinel {
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(expectRemoved))
			continue
		}

		// Otherwise, dequeue and return the byte task
		dequeued = expectRemoved.task
		success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)), unsafe.Pointer(expectSentinel), unsafe.Pointer(expectRemoved)) // dequeue

	}

	return dequeued

}

func (q *lfdeque) PopTop() (task *ImageTask) {
	var dequeued *ImageTask
	var expectSentinel, expectRemoved, expectHead *node
	success := false
	for !success {
		expectSentinel = q.tail
		expectRemoved = expectSentinel.prev
		expectHead = q.head

		// If not at the head then try again
		if q.tail != expectSentinel {
			continue
		}

		// Signal that queue is empty when the sentinel node is reached
		if expectRemoved == nil {
			return nil
		}

		// Help tail along if it is behind and try again
		if expectHead == expectSentinel {
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)), unsafe.Pointer(expectHead), unsafe.Pointer(expectRemoved))
			continue
		}

		// Otherwise, dequeue and return the byte task
		dequeued = expectRemoved.task
		success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectSentinel), unsafe.Pointer(expectRemoved)) // dequeue

	}

	return dequeued

}

// func (q *lfdeque) PopTop() *ImageTask {
// 	var dequeued *ImageTask
// 	var expectTail, expectPrevTail *node

// 	success := false
// 	for !success {
// 		expectTail = q.tail
// 		if expectTail == q.head { // Check if the queue is empty or only has the sentinel node.
// 			return nil // The queue is empty.
// 		}
// 		// identify the node right before expectTail (this requires a traversal from head in a singly linked list, which is inefficient)
// 		var prev *node = nil
// 		for node := q.head; node != expectTail && node.next != expectTail; node = node.next {
// 			prev = node
// 		}
// 		expectPrevTail = prev

// 		// If the queue was modified during traversal, retry.
// 		if q.tail != expectTail {
// 			continue
// 		}

// 		dequeued = expectTail.task // Prepare to return the task at the tail.
// 		// Attempt to adjust the tail to the previous node. If the CAS operation fails, retry.
// 		success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(expectPrevTail))

// 		// If successful, we need to adjust the next pointer of the new tail (expectPrevTail) to nil, effectively removing the old tail from the queue.
// 		if success && expectPrevTail != nil {
// 			expectPrevTail.next = nil
// 		}
// 	}

// 	return dequeued
// }

func (q *lfdeque) Steal(thiefID int, workPools []WorkPool) bool {
	for i, pool := range workPools {
		if i != thiefID { // Don't steal from itself
			task := pool.PopTop() // Attempt to steal from the top of other pools
			if task != nil {
				q.PushBottom(task) // If successful, push the stolen task to the bottom of our deque
				fmt.Printf("Work pool %d stole task from work pool %d.\n", thiefID, i)
				return true
			}
		}
	}
	return false
}
