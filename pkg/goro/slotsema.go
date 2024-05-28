package goro

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	ErrNoSlotsAvailable    = errors.New("no slots available")
	ErrSlotAlreadyReleased = errors.New("slot already released")
)

type slot struct {
	parentSema *SlottedSemaphore
}

func (s *slot) Release() error {
	if s.parentSema == nil {
		logrus.Errorf("slot already released")
		return ErrSlotAlreadyReleased
	}

	if err := s.parentSema.Release(s); err != nil {
		logrus.Fatalf("failed to release slot: %s", err.Error())
	}

	logrus.Tracef("released slot %v", s)
	s.parentSema = nil

	return nil
}

func (s *slot) IsReleased() bool {
	return s.parentSema == nil
}

type SlottedSemaphore struct {
	sema            []*slot
	lock            *sync.Mutex
	acquisitionCond *sync.Cond
}

func NewSlottedSemaphore(limit int) *SlottedSemaphore {
	lock := &sync.Mutex{}
	return &SlottedSemaphore{
		sema:            make([]*slot, limit),
		lock:            lock,
		acquisitionCond: sync.NewCond(lock),
	}
}

func (ss *SlottedSemaphore) findFreeSlot() int {
	for i, slot := range ss.sema {
		if slot == nil {
			logrus.Tracef("first free slot at %d", i)
			return i
		}
	}

	return -1
}

func (ss *SlottedSemaphore) Acquire() (*slot, error) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	slotIdx := ss.findFreeSlot()
	if slotIdx == -1 {
		return nil, ErrNoSlotsAvailable
	}

	slot := &slot{
		parentSema: ss,
	}
	ss.sema[slotIdx] = slot
	logrus.Tracef("slot %d acquired", slotIdx)

	return slot, nil
}

func (ss *SlottedSemaphore) AcquireBlocking() *slot {
	ss.lock.Lock()

	for {
		slotIdx := ss.findFreeSlot()
		if slotIdx != -1 {
			slot := &slot{
				parentSema: ss,
			}
			ss.sema[slotIdx] = slot
			logrus.Tracef("slot %d acquired (blocking)", slotIdx)
			ss.lock.Unlock()

			return slot
		}

		logrus.Trace("no slots available, sleeping for release")
		ss.acquisitionCond.Wait()
	}
}

func (ss *SlottedSemaphore) Release(s *slot) error {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for i, slot := range ss.sema {
		if slot == s {
			logrus.Tracef("releasing slot at index %d", i)
			ss.sema[i] = nil
			ss.acquisitionCond.Signal()
			return nil
		}
	}

	return ErrSlotAlreadyReleased
}
