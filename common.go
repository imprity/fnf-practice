package main

import (
	"golang.org/x/exp/constraints"
)

func BoolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func IntToBool[N constraints.Integer](n N) bool {
	if n == 0 {
		return false
	} else {
		return true
	}
}

func AbsI[N constraints.Signed](n N) N {
	if n < 0 {
		return n * -1
	}
	return n
}

func SameSign[N constraints.Signed](n1, n2 N) bool {
	return (n1 < 0) == (n2 < 0)
}

func Clamp[N constraints.Integer | constraints.Float](n, minN, maxN N) N {
	n = min(n, maxN)
	n = max(n, minN)

	return n
}

func Lerp[F constraints.Float](a, b, t F) F{
	return a + (b-a) * t
}

type CircularQueue[T any] struct {
	Length int
	Start  int
	Data   []T
}

func (q *CircularQueue[T]) IsFull() bool {
	return q.Length >= len(q.Data)
}

func (q *CircularQueue[T]) IsEmpty() bool {
	return q.Length <= 0
}

func (q *CircularQueue[T]) Enqueue(item T) {
	isFull := q.IsFull()
	if isFull {
		q.Start += 1
		q.Start = q.Start % q.Length
	} else {
		q.Length += 1
	}

	index := (q.Start + q.Length - 1) % len(q.Data)
	q.Data[index] = item
}

func (q *CircularQueue[T]) Dequeue() T {
	if q.Length <= 0 {
		panic("CircularQueue:Dequeue: Dequeue on empty queue")
	}

	q.Start %= len(q.Data)
	returnIndex := q.Start
	q.Start += 1

	return q.Data[returnIndex]
}

func (q *CircularQueue[T]) At(index int) T {
	return q.Data[(q.Start+index)%q.Length]
}

type ReadChannel[T any] struct {
	RequestChannel chan bool
	DataChannel    chan T
}

func (rc ReadChannel[T]) RequestRead() {
	rc.RequestChannel <- true
}

func (rc ReadChannel[T]) Read() T {
	return <-rc.DataChannel
}

type ReadManyChannel[T any] struct {
	RequestChannel chan bool
	SizeChannel    chan int
	DataChannel    chan T
}

func (rm ReadManyChannel[T]) RequestRead() {
	rm.RequestChannel <- true
}

func (rm ReadManyChannel[T]) ReadSize() int {
	return <-rm.SizeChannel
}

func (rm ReadManyChannel[T]) Read() T {
	return <-rm.DataChannel
}

