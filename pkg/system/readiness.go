package system

import (
	"context"
	"fmt"
	"time"
)

type ReadinessCheckFunc func(context.Context) (string, error)

var readinessChecks = map[string]ReadinessCheckFunc{}

func AddReadinessCheck(name string, checkFunc ReadinessCheckFunc) {
	readinessChecks[name] = checkFunc
}

func RunReadinessChecks(parentCtx context.Context) (string, map[string]string) {
	ctx, cancel := context.WithDeadline(parentCtx, time.Now().Add(15*time.Second))
	defer cancel()

	status := StatusOk
	components := make(map[string]string)

	for name, checkFn := range readinessChecks {
		if message, err := checkFn(ctx); err != nil {
			status = StatusUnhealthy
			components[name] = err.Error()
		} else {
			if message == "" {
				message = StatusOk
			} else {
				message = fmt.Sprintf("%s: %s", StatusOk, message)
			}
			components[name] = message
		}
	}

	return status, components
}
