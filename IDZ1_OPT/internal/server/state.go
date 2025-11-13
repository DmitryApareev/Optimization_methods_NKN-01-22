package server

import (
	"context"
	"sync"
	"time"

	"idz1_opt/internal/optimizer"
)

// параметры запуска метода
type RunParams struct {
	Func    string  `json:"func"`
	A       float64 `json:"a"`
	B       float64 `json:"b"`
	Eps     float64 `json:"eps"`
	Delta   float64 `json:"delta"`
	MaxIter int     `json:"maxIter"`
}

// состояние одного запуска
type RunState struct {
	ID        string
	Params    RunParams
	CreatedAt time.Time

	LastIter optimizer.Iter
	Iters    []optimizer.Iter

	Err    string
	Done   bool
	Cancel context.CancelFunc
}

var (
	runsMu sync.Mutex
	runs   = map[string]*RunState{}
)

func saveRun(rs *RunState) {
	runsMu.Lock()
	defer runsMu.Unlock()
	runs[rs.ID] = rs
}

func getRun(id string) *RunState {
	runsMu.Lock()
	defer runsMu.Unlock()
	return runs[id]
}
