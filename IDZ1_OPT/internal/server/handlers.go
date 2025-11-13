package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"idz1_opt/internal/optimizer"
	"idz1_opt/internal/sse"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// StartRun запускает новый процесс минимизации
func StartRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "только POST", http.StatusMethodNotAllowed)
		return
	}

	var p RunParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "ошибка JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if p.MaxIter <= 0 {
		p.MaxIter = 200
	}
	if p.Eps <= 0 {
		p.Eps = 1e-5
	}
	if p.Delta <= 0 {
		p.Delta = p.Eps / 2
	}
	if !(p.A < p.B) {
		http.Error(w, "требуется a < b", http.StatusBadRequest)
		return
	}

	f, err := optimizer.NewEvalFunc(p.Func)
	if err != nil {
		http.Error(w, "ошибка в выражении функции: "+err.Error(), http.StatusBadRequest)
		return
	}

	// предварительно считаем значения функции для графика
	const n = 400
	xs := make([]float64, n)
	ys := make([]float64, n)
	h := (p.B - p.A) / float64(n-1)
	for i := 0; i < n; i++ {
		x := p.A + float64(i)*h
		y, err := f.Eval(x)
		if err != nil || math.IsNaN(y) || math.IsInf(y, 0) {
			y = math.NaN()
		}
		xs[i], ys[i] = x, y
	}

	id := uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())
	rs := &RunState{
		ID:        id,
		Params:    p,
		CreatedAt: time.Now(),
		Cancel:    cancel,
	}
	saveRun(rs)

	// асинхронный запуск оптимизации
	go func() {
		// стартовое событие
		startMsg, _ := json.Marshal(map[string]any{
			"type": "start",
			"id":   id,
		})
		sse.Publish(id, string(startMsg))

		onIter := func(it optimizer.Iter) error {
			select {
			case <-ctx.Done():
				return optimizer.ErrStopped
			default:
			}

			rs.LastIter = it
			rs.Iters = append(rs.Iters, it)

			payload := map[string]any{
				"type": "iter",
				"iter": it,
			}
			msg, _ := json.Marshal(payload)
			sse.Publish(id, string(msg))
			return nil
		}

		last, err := optimizer.Dichotomy(
			f,
			p.A, p.B,
			p.Eps, p.Delta,
			p.MaxIter,
			onIter,
		)

		if err != nil {
			if err == optimizer.ErrStopped || err == context.Canceled {
				stopMsg, _ := json.Marshal(map[string]any{
					"type": "stopped",
				})
				sse.Publish(id, string(stopMsg))
				return
			}

			rs.Err = "ошибка при вычислении: " + err.Error()
			errMsg, _ := json.Marshal(map[string]any{
				"type": "error",
				"err":  rs.Err,
			})
			sse.Publish(id, string(errMsg))
			return
		}

		rs.Done = true
		rs.LastIter = last

		doneMsg, _ := json.Marshal(map[string]any{
			"type": "done",
			"x":    last.XMid,
			"fx":   last.FXMid,
		})
		sse.Publish(id, string(doneMsg))
	}()

	resp := map[string]any{
		"id": id,
		"xs": xs,
		"ys": ys,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// StopRun — прерывание процесса минимизации
func StopRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "только POST", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "требуется id", http.StatusBadRequest)
		return
	}

	rs := getRun(id)
	if rs == nil {
		http.Error(w, "неизвестный id", http.StatusNotFound)
		return
	}

	if rs.Cancel != nil {
		rs.Cancel()
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExportCSV — экспорт итераций в CSV
func ExportCSV(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "требуется id", http.StatusBadRequest)
		return
	}

	rs := getRun(id)
	if rs == nil {
		http.Error(w, "неизвестный id", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=iterations_"+id+".csv")

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"k", "a", "b", "mid", "f(mid)", "b-a"})

	for _, it := range rs.Iters {
		_ = cw.Write([]string{
			strconv.Itoa(it.K),
			fmtFloat(it.A),
			fmtFloat(it.B),
			fmtFloat(it.XMid),
			fmtFloat(it.FXMid),
			fmtFloat(it.Len),
		})
	}
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', 16, 64)
}

// Stream — SSE-стрим итераций
func Stream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "требуется id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch, cancel := sse.Subscribe(id)
	defer cancel()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "event: msg\n")
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
