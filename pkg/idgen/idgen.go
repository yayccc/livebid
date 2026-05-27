package idgen

import (
	"sync"
	"time"
)

const (
	epochMillis  int64 = 1704067200000 // 2024-01-01 00:00:00 UTC
	sequenceBits       = 12
	workerBits         = 10
	maxSequence        = int64(1<<sequenceBits - 1)
	maxWorkerID        = int64(1<<workerBits - 1)
	workerShift        = sequenceBits
	timeShift          = workerBits + sequenceBits
)

type Generator struct {
	mu        sync.Mutex
	workerID  int64
	lastMilli int64
	sequence  int64
}

func New(workerID int64) *Generator {
	if workerID < 0 || workerID > maxWorkerID {
		workerID = 0
	}
	return &Generator{workerID: workerID}
}

func (g *Generator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := currentMillis()
	if now < g.lastMilli {
		now = g.lastMilli
	}

	if now == g.lastMilli {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			now = waitNextMillis(g.lastMilli)
		}
	} else {
		g.sequence = 0
	}

	g.lastMilli = now
	return ((now - epochMillis) << timeShift) | (g.workerID << workerShift) | g.sequence
}

func currentMillis() int64 {
	return time.Now().UnixMilli()
}

func waitNextMillis(last int64) int64 {
	for {
		now := currentMillis()
		if now > last {
			return now
		}
		time.Sleep(time.Millisecond)
	}
}
