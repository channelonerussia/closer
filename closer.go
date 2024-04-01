// Package closer нужен для централизованного закрытия всего, что требует закрытия (бд, пулы и т.д., и т.п.)
package closer

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

var globalCloser = New(syscall.SIGINT, syscall.SIGTERM)

// Add нужна для добавления функций закрытия
func Add(f ...Callback) {
	globalCloser.add(f...)
}

// Wait нужна для постановки closer'а в ожидание сигнала о закрытии
func Wait() {
	globalCloser.wait()
}

// CloseAll нужна для вызова всех закрывающих коллбеков
func CloseAll() {
	globalCloser.closeAll()
}

// New возвращает новый Closer, если приходит один из []os.Signal определенных в New,то Closer вызовет CloseAll.
func New(sig ...os.Signal) Closer {
	c := &closer{done: make(chan struct{})}
	if len(sig) > 0 {
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, sig...)
			<-ch
			signal.Stop(ch)
			c.closeAll()
		}()
	}
	return c
}

// Add добавляет коллбек в слайс коллбеков. Безопасна для конкурентного режима.
func (c *closer) add(f ...Callback) {
	c.mu.Lock()
	c.funcs = append(c.funcs, f...)
	c.mu.Unlock()
}

// Wait блокирует выход из программы, пока все коллбеки не будут выполнены
func (c *closer) wait() {
	<-c.done
}

// CloseAll вызывает все коллбеки
func (c *closer) closeAll() {
	c.once.Do(func() {
		defer close(c.done)

		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		// вызываем все функции асинхронно
		errs := make(chan error, len(funcs))
		for _, f := range funcs {
			go func(f func() error) {
				errs <- f()
			}(f)
		}

		for i := 0; i < cap(errs); i++ {
			if err := <-errs; err != nil {
				log.Println("error returned from Closer: %w", err)
			}
		}
	})
}
