package closer

import "sync"

// Closer нужен для создания глобального closer'а
type Closer interface {
	add(f ...Callback)
	wait()
	closeAll()
}

type Callback = func() error

type closer struct {
	mu    sync.Mutex
	once  sync.Once
	done  chan struct{}
	funcs []func() error
}
