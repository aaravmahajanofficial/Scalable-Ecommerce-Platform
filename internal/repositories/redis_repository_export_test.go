package repository

import "time"

// Export timeNow for testing
func SetTimeNow(f func() time.Time) {
	timeNow = f
}
