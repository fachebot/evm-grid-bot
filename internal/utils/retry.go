package utils

import (
	"errors"
	"strings"
	"time"
)

func Retry(attempts int, sleep time.Duration, fn func() error) error {
	var errList []string
	var err error

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		errList = append(errList, err.Error())
		time.Sleep(sleep)
	}

	return errors.New("retry failed: " + strings.Join(errList, "; "))
}
