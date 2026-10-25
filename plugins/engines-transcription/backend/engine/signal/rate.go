// Package signal holds what every stage of audio processing agrees on: the
// sample rate the models hear, how long a count of samples lasts, and how many
// goroutines share the work.
package signal

// SampleRate is the rate, in hertz, of the mono audio every model is given.
const SampleRate = 16000

// GetMillis is how long a count of samples at one rate lasts.
func GetMillis(count, rate int) int {
	if rate <= 0 {
		return 0
	}
	return int(int64(count) * 1000 / int64(rate))
}
