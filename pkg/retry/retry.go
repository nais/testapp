package retry

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type ContextConfig struct {
	Ctx     context.Context
	max     time.Duration
	timeout time.Duration
}

func NewContextConfig(ctx context.Context, max, timeout int) *ContextConfig {
	retryCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(max),
	)

	defer cancel()

	return &ContextConfig{
		Ctx:     retryCtx,
		max:     time.Duration(max),
		timeout: time.Duration(timeout),
	}
}

func Do(config *ContextConfig, statement func() error, errorOK func(err error) bool) error {
	var err error
	attempt := 0
	for {
		err = statement()
		if err == nil || errorOK(err) {
			return nil
		}

		select {
		case <-config.Ctx.Done():
			return fmt.Errorf("gave up retrying after %d seconds: last error: %v", config.max, err)
		case <-time.After(config.timeout * time.Second):
			attempt++
			log.Info("retrying %d", attempt)
		}
	}
}
