package common

import (
	"time"
)

// WaitForCompletion attempts task at repeat intervals until timeout or success (bool return is true)
func WaitForCompletion(repeat, timeout time.Duration, task func() (interface{},bool)) (interface{},bool) {
	start := time.Now()
	for time.Since(start) < timeout {
		res, completed := task()
		if completed {
			return res, true
		}
		time.Sleep(repeat)
	}
	return nil, false
}
