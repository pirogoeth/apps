package goro

import (
	"testing"
	"time"

	"github.com/pirogoeth/apps/pkg/logging"
)

func TestMain(m *testing.M) {
	logging.Setup()

	m.Run()
}

func TestSlottedSemaphore(t *testing.T) {
	ss := NewSlottedSemaphore(1)
	results := make([]int, 2)
	doneCh := make(chan bool)

	go func() {
		t.Log("Acquiring slot 1")
		slot := ss.AcquireBlocking()
		t.Logf("Acquired slot 1: %v", slot)
		defer slot.Release()

		time.Sleep(1 * time.Second)
		results[0] = 1
		doneCh <- true
	}()

	go func() {
		t.Log("Acquiring slot 2")
		slot := ss.AcquireBlocking()
		t.Logf("Acquired slot 2: %v", slot)
		defer slot.Release()

		results[1] = 2
		doneCh <- true
	}()

	<-doneCh
	<-doneCh

	if results[0] != 1 || results[1] != 2 {
		t.Fail()
	}
}

func TestSlottedSemaphoreDoubleRelease(t *testing.T) {
	ss := NewSlottedSemaphore(1)
	slot := ss.AcquireBlocking()
	slot.Release()
	err := slot.Release()
	if err != ErrSlotAlreadyReleased {
		t.Fail()
	}
}

func TestSlottedSemaphoreAsyncAcquire(t *testing.T) {
	ss := NewSlottedSemaphore(1)
	slot, err := ss.Acquire()
	if err != nil {
		t.Fail()
	}

	go func() {
		time.Sleep(1 * time.Second)
		slot.Release()
	}()

	for {
		if slot.IsReleased() {
			break
		}
	}
}

func TestSlottedSemaphoreGoroWait(t *testing.T) {
	ss := NewSlottedSemaphore(1)
	slot := ss.AcquireBlocking()
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			_, err := ss.Acquire()
			if err == nil {
				t.Errorf("should not be able to acquire a slot")
			}

			t.Log("Waiting for slot")
			time.Sleep(500 * time.Millisecond)
		}

		done <- true
	}()

	<-done
	slot.Release()

	t.Log("Waiting goro closed")
}
