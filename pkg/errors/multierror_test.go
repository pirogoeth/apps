package errors_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/pirogoeth/apps/pkg/errors"
)

var TestMultiErrorOutput = regexp.MustCompile(`\*\*\* 2 errors have occurred:

----------------------------------------
\| Error #0
\| at \w+:\d+ [^\n]+
\|
\| a
\\---------------------------------------
\| Error #1
\| at \w+:\d+ [^\n]+
\|
\| b
\\---------------------------------------
`)

func TestMultiError(t *testing.T) {
	m := new(errors.MultiError)
	m.Add(fmt.Errorf("a"))
	m.Add(fmt.Errorf("b"))

	if err := m.Error(); TestMultiErrorOutput.Match([]byte(err)) {
		t.Errorf(`output does not match. expected regex (equals signs as delimiters):
=======================================
%s
=======================================
      >>> but got:
=======================================
%s
=======================================
`, TestMultiErrorOutput, []byte(err))
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
