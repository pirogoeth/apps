package errors

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

var _ error = (*MultiError)(nil)

type errLoc struct {
	file     string
	lineNo   int
	funcName string
}

func (e *errLoc) String() string {
	if e == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%s:%d %s()", e.file, e.lineNo, e.funcName)
}

func errLocFrom(file string, lineNo int, pc uintptr) *errLoc {
	return &errLoc{
		file:     file,
		lineNo:   lineNo,
		funcName: runtime.FuncForPC(pc).Name(),
	}
}

type MultiError struct {
	errs    []error
	errLocs []*errLoc
	mu      sync.Mutex
}

func (m *MultiError) Add(err error) {
	if err == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pc, file, lineNo, ok := runtime.Caller(1)
	if !ok {
		m.errLocs = append(m.errLocs, nil)
	} else {
		m.errLocs = append(m.errLocs, errLocFrom(file, lineNo, pc))
	}

	m.errs = append(m.errs, err)
}

func (m *MultiError) ToError() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.errs) > 0 {
		return m
	}

	return nil
}

func (m *MultiError) Error() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	builder := new(strings.Builder)
	fmt.Fprintf(builder, "*** %d errors have occurred:\n\n", len(m.errs))
	fmt.Fprintf(builder, "%s\n", strings.Repeat("-", 40))
	for idx, err := range m.errs {
		fmt.Fprintf(builder, "| Error #%d\n", idx)
		fmt.Fprintf(builder, "| at %s\n|\n", m.errLocs[idx])
		for _, errLine := range strings.Split(err.Error(), "\n") {
			fmt.Fprintf(builder, "| %s\n", errLine)
			fmt.Fprintf(builder, "\\%s\n", strings.Repeat("-", 39))
		}
	}

	return builder.String()
}
