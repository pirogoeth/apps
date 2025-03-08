package errors_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pirogoeth/apps/pkg/errors"
)

const TestMultiErrorOutput = `*** 2 errors have occurred:

----------------------------------------
| Error #0
|
| a
\---------------------------------------
| Error #1
|
| b
\---------------------------------------
`

func TestMultiError(t *testing.T) {
	m := new(errors.MultiError)
	m.Add(fmt.Errorf("a"))
	m.Add(fmt.Errorf("b"))

	if err := m.Error(); err != TestMultiErrorOutput {
		t.Errorf(`output does not match. expected (equals signs as delimiters):
=======================================
%s
=======================================
      >>> but got:
=======================================
%s
=======================================
`, TestMultiErrorOutput, err)
	}
}

func TestMultiErrorRace(t *testing.T) {
	m := new(errors.MultiError)
	ctx, cancel := context.WithCancel(context.Background())
	for i := range []uint8{1, 2} {
		go func() {
			for {
				select {
				case <-ctx.Done():
					m.Add(fmt.Errorf("Goro %d has been cancelled!", i))
				}
			}
		}()
	}

	cancel()
}
