package sse

import "sync"

// простой hub для SSE по runID

var (
	mu    sync.Mutex
	conns = map[string][]chan string{}
)

// Subscribe подписывает клиента на id, возвращает канал и функцию-unsubscribe
func Subscribe(id string) (chan string, func()) {
	ch := make(chan string, 16)

	mu.Lock()
	conns[id] = append(conns[id], ch)
	mu.Unlock()

	cancel := func() {
		mu.Lock()
		defer mu.Unlock()
		list := conns[id]
		for i, c := range list {
			if c == ch {
				conns[id] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}

	return ch, cancel
}

// Publish отсылает сообщение всем подписчикам runID
func Publish(id, msg string) {
	mu.Lock()
	list := append([]chan string(nil), conns[id]...)
	mu.Unlock()

	for _, ch := range list {
		select {
		case ch <- msg:
		default:
			// игнорируем, если канал забит
		}
	}
}
