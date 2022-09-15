package retry

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

type ContextConfig struct {
	name    string
	Ctx     context.Context
	max     time.Duration
	timeout time.Duration
}

func NewContextConfig(ctx context.Context, name string, max, timeout int) *ContextConfig {
	m := time.Duration(max) * time.Second
	t := time.Duration(timeout) * time.Second

	retryCtx, cancel := context.WithTimeout(
		ctx,
		m,
	)

	defer cancel()

	return &ContextConfig{
		name:    name,
		Ctx:     retryCtx,
		max:     m,
		timeout: t,
	}
}

func Do(config *ContextConfig, statement func() error, errorOK func(err error) bool) error {
	var err error
	attempt := 1
	for {
		err = statement()
		if err == nil || errorOK(err) {
			log.Infof("initial test for %s statement succeeded after %d attempts", config.name, attempt)
			return nil
		}

		select {
		case <-config.Ctx.Done():
			return fmt.Errorf("gave up retrying after %f seconds: last error: %v", math.Trunc(config.max.Seconds()), err)
		case <-time.After(config.timeout):
			attempt++
			log.Infof("retrying %d", attempt)
		}
	}
}
