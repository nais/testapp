package retry

import (
	"context"
	"fmt"
	"time"
)

type ContextConfig struct {
	ctx     context.Context
	max     time.Duration
	timeout time.Duration
	Cancel  context.CancelFunc
}

func NewContextConfig(ctx context.Context, max, timeout int) *ContextConfig {
	retryCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(max),
	)

	return &ContextConfig{
		ctx:     retryCtx,
		max:     time.Duration(max),
		timeout: time.Duration(timeout),
		Cancel:  cancel,
	}
}

func Do(config *ContextConfig, errorFunc func() error, errorOK func(err error) bool) error {
	var err error
	for {
		err = errorFunc()
		if err == nil || errorOK(err) {
			return nil
		}

		select {
		case <-config.ctx.Done():
			return fmt.Errorf("gave up retrying after %d seconds: last error: %v", config.max, err)
		case <-time.After(config.timeout * time.Second):
		}
	}
}
