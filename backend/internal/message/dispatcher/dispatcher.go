package dispatcher

import (
	"context"
	"log"
	"sync"

	"d-im/pkg/model"
	"d-im/pkg/snowflake"

	"d-im/internal/message/repository"
)

// Dispatcher 消息分发器
type Dispatcher struct {
	repo    *repository.MessageRepo
	idGen   *snowflake.Generator
	workers int
	queue   chan *dispatchTask
	wg      sync.WaitGroup
}

type dispatchTask struct {
	Msg        *model.Message
	TargetUIDs []string
}

// NewDispatcher 创建分发器
func NewDispatcher(repo *repository.MessageRepo, idGen *snowflake.Generator, workers int) *Dispatcher {
	if workers <= 0 {
		workers = 4
	}
	return &Dispatcher{
		repo:    repo,
		idGen:   idGen,
		workers: workers,
		queue:   make(chan *dispatchTask, 1024),
	}
}

// Start 启动工作协程
func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker(ctx, i)
	}
	log.Printf("[dispatcher] started with %d workers", d.workers)
}

// Stop 停止分发器
func (d *Dispatcher) Stop() {
	close(d.queue)
	d.wg.Wait()
	log.Println("[dispatcher] stopped")
}

// Dispatch 投递分发任务到队列（非阻塞）
func (d *Dispatcher) Dispatch(msg *model.Message, targetUIDs []string) {
	select {
	case d.queue <- &dispatchTask{Msg: msg, TargetUIDs: targetUIDs}:
	default:
		log.Printf("[dispatcher] queue full, dropping task: msg_id=%s", msg.MsgID)
	}
}

// worker 工作协程
func (d *Dispatcher) worker(ctx context.Context, id int) {
	defer d.wg.Done()
	for task := range d.queue {
		select {
		case <-ctx.Done():
			return
		default:
		}

		mailboxes := make([]*model.UserMailbox, len(task.TargetUIDs))
		for i, uid := range task.TargetUIDs {
			mailboxes[i] = &model.UserMailbox{
				UID:        uid,
				ChatID:     task.Msg.ChatID,
				MsgID:      task.Msg.MsgID,
				MessageSeq: task.Msg.Seq,
				SeqID:      d.idGen.Generate(),
				Status:     task.Msg.Status,
			}
		}

		if err := d.repo.BatchInsertToMailbox(ctx, mailboxes); err != nil {
			log.Printf("[dispatcher] worker-%d batch insert failed: %v", id, err)
		}
	}
}
