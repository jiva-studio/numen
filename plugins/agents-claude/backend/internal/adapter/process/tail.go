package process

import "sync"

// maxStderr is how much of the end of the standard error of a program is kept.
const maxStderr = 64 << 10

// tailBuffer keeps the last maxStderr bytes written to it.
type tailBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range p {
		if len(b.data) == maxStderr {
			b.data = b.data[1:]
		}
		b.data = append(b.data, c)
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
