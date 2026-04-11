package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type tailManager struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	states []*tailState
	dst    io.Writer
}

type tailState struct {
	path   string
	mu     sync.Mutex
	offset int64
}

func startTailers(parent context.Context, paths []string, dst io.Writer, verbose bool) *tailManager {
	ctx, cancel := context.WithCancel(parent)
	m := &tailManager{cancel: cancel, dst: dst}
	for _, path := range paths {
		state := &tailState{path: path}
		m.states = append(m.states, state)
		m.wg.Add(1)
		go func(s *tailState) {
			defer m.wg.Done()
			tailFile(ctx, s, dst, verbose)
		}(state)
	}
	return m
}

func (m *tailManager) StopAndFlush() {
	if m == nil {
		return
	}
	m.cancel()
	m.wg.Wait()
	for _, state := range m.states {
		flushTailState(state, m.dst)
	}
}

func tailFile(ctx context.Context, state *tailState, dst io.Writer, verbose bool) {
	var file *os.File
	closeFile := func() {
		if file != nil {
			_ = file.Close()
			file = nil
		}
	}
	defer closeFile()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if file == nil {
			f, err := os.Open(state.path)
			if err != nil {
				continue
			}
			file = f
			if _, err := file.Seek(state.currentOffset(), io.SeekStart); err != nil {
				closeFile()
				continue
			}
			if verbose {
				fmt.Fprintf(dst, "[tini-win] tailing file: %s\n", state.path)
			}
		}

		if err := streamNewBytes(file, state, dst); err != nil {
			closeFile()
			continue
		}
	}
}

func flushTailState(state *tailState, dst io.Writer) {
	file, err := os.Open(state.path)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Seek(state.currentOffset(), io.SeekStart)
	_, _ = io.Copy(dst, file)
}

func streamNewBytes(file *os.File, state *tailState, dst io.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	offset := state.currentOffset()
	if info.Size() < offset {
		offset = 0
		state.setOffset(0)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	if info.Size() == offset {
		return nil
	}
	buf := make([]byte, info.Size()-offset)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	if n > 0 {
		state.addOffset(int64(n))
		_, _ = dst.Write(buf[:n])
	}
	return nil
}

func (s *tailState) currentOffset() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offset
}

func (s *tailState) setOffset(v int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offset = v
}

func (s *tailState) addOffset(v int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offset += v
}
