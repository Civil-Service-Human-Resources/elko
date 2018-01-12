// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package plugin

import (
	"sync"

	"github.com/tav/v8worker"
)

type Handler struct {
	mux  sync.RWMutex
	name string
	w    *v8worker.Worker
}

func (h *Handler) Configure(name string, config string) {
	h.w.SendSync()
}

func (h *Handler) Update(code string) error {
	w := v8worker.New(noop, noopSync)
	err := w.Load("plugin/"+h.name+".js", code)
	if err != nil {
		return err
	}
	existing := h.w
	h.mux.Lock()
	h.w = w
	h.mux.Unlock()
	existing.TerminateExecution()
	existing.Dispose()
	return nil
}

func noop(msg string) {
}

func noopSync(msg string) string {
}

func New(name string, code string) (*Handler, error) {
	w := v8worker.New(noop, noopSync)
	err := w.Load("plugin/"+name+".js", code)
	if err != nil {
		return nil, err
	}
	return &Handler{
		name: name,
		w:    w,
	}, nil
}
