package snowflake

import (
	"fmt"
	"sync"
	"time"
)

// 雪花ID epoch: 2024-01-01 00:00:00 UTC (毫秒级)
const epoch = 1704067200000

const (
	workerIDBits     = 5
	datacenterIDBits = 5
	sequenceBits     = 12

	maxWorkerID     = -1 ^ (-1 << workerIDBits)
	maxDatacenterID = -1 ^ (-1 << datacenterIDBits)

	workerIDShift     = sequenceBits
	datacenterIDShift = sequenceBits + workerIDBits
	timestampShift    = sequenceBits + workerIDBits + datacenterIDBits

	sequenceMask = -1 ^ (-1 << sequenceBits)
)

// Generator 雪花ID生成器
type Generator struct {
	mu            sync.Mutex
	workerID      int64
	datacenterID  int64
	sequence      int64
	lastTimestamp int64
}

// Config 雪花ID配置
type Config struct {
	WorkerID     int64 `yaml:"worker_id"`
	DatacenterID int64 `yaml:"datacenter_id"`
}

// NewGenerator 创建雪花ID生成器
func NewGenerator(cfg Config) (*Generator, error) {
	if cfg.WorkerID < 0 || cfg.WorkerID > maxWorkerID {
		return nil, fmt.Errorf("worker_id must be between 0 and %d", maxWorkerID)
	}
	if cfg.DatacenterID < 0 || cfg.DatacenterID > maxDatacenterID {
		return nil, fmt.Errorf("datacenter_id must be between 0 and %d", maxDatacenterID)
	}

	return &Generator{
		workerID:     cfg.WorkerID,
		datacenterID: cfg.DatacenterID,
	}, nil
}

// Generate 生成下一个ID
func (g *Generator) Generate() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < g.lastTimestamp {
		// 时钟回拨，等待追上
		diff := g.lastTimestamp - now
		if diff <= 5 {
			time.Sleep(time.Duration(diff) * time.Millisecond)
			now = time.Now().UnixMilli()
		}
	}

	if now == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & sequenceMask
		if g.sequence == 0 {
			// 序列号耗尽，等待下一毫秒
			for now <= g.lastTimestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = now

	return ((now - epoch) << timestampShift) |
		(g.datacenterID << datacenterIDShift) |
		(g.workerID << workerIDShift) |
		g.sequence
}

// GenerateString 生成字符串格式的ID
func (g *Generator) GenerateString() string {
	return fmt.Sprintf("%d", g.Generate())
}
