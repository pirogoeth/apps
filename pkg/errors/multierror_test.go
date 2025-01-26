package errors_test

import (
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
