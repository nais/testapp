package util

import (
	"context"
	"fmt"
	"time"
)

func Retry(ctx context.Context, f func() error, errorOK func(err error) bool, retries int) error {
	var err error
	for tries := 0; tries < retries; tries++ {
		err = f()
		if err != nil {
			if errorOK(err) {
				return nil
			}

			time.Sleep(500 * time.Duration(tries+1) * time.Millisecond)
			continue
		}
		break
	}

	if err != nil {
		<-ctx.Done()
		return fmt.Errorf("gave up retrying after %d attempts: last error: %v", retries, err)
	}

	return nil
}
