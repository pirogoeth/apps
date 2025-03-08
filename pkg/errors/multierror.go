package errors

import (
	"fmt"
	"strings"
	"sync"
)

var _ error = (*MultiError)(nil)

type MultiError struct {
	errs []error
	mu   sync.Mutex
}

func (m *MultiError) Add(err error) {
	if err == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

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
		fmt.Fprintf(builder, "| Error #%d\n|\n", idx)
		for _, errLine := range strings.Split(err.Error(), "\n") {
			fmt.Fprintf(builder, "| %s\n", errLine)
			fmt.Fprintf(builder, "\\%s\n", strings.Repeat("-", 39))
		}
	}

	return builder.String()
}
