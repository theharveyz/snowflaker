package snowflake

import (
	"errors"
	"sync"
	"time"
)

// worker id 固定
// 时间戳 uint(41) + uint(10) + uint(12)
// 时间戳部分精确到毫秒, 如果在一毫秒内把sequence id(4095个)用完了,则等到下一毫秒再获取,
// 	并且sequence id从0开始算

// 约束
var _ Snowflaker = (*worker)(nil)

// 常量
const (
	workerIDBits     = 10
	sequenceBits     = 12
	maxWorkerID      = (1 << workerIDBits) - 1
	beginMilliSecond = 1510687485246 // 决定id最大值
	maxSequence      = (1 << sequenceBits) - 1
)

var (
	invalidWorkIDError      = errors.New("invalid worker id")
	clockMovedBackwardError = errors.New("Clock moved backwards, Refuse gen id")
	invalidIDError          = errors.New("invalid id")
)

// Snowflaker interface
type Snowflaker interface {
	NextID() (int64, error)
}

// New return a Snowflaker implement
func New(wid int64) (*worker, error) {
	if wid > maxWorkerID {
		return nil, invalidWorkIDError
	}
	w := &worker{
		workerID: wid,
	}
	// 初始化
	return w, nil
}

type worker struct {
	workerID        int64        // worker id
	sequence        int64        // 自增序列:从0开始计算
	lastMillisecond int64        // 最新的毫秒数
	lock            sync.RWMutex // 这里不用指针, 限制方法的receiver必须为指针
}

func (w *worker) WorkerID() int64 {
	return w.workerID
}

func (w *worker) Sequence() int64 {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.sequence
}

func (w *worker) LastMillisecond() int64 {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.lastMillisecond
}

func (w *worker) NextID() (int64, error) {
	// 执行的串行性(获取全局ID整个操作本身就应该是串行)
	w.lock.Lock()
	defer w.lock.Unlock()
	// 获取时间戳, 如果小于当前时间戳则等待
	now := getUnixMillisecond()
	if now < w.lastMillisecond {
		return 0, clockMovedBackwardError
	}
	if now > w.lastMillisecond {
		w.lastMillisecond = now
		w.sequence = 0
	} else {
		w.sequence = (w.sequence + 1) & maxSequence
		// 如果达到最大值,则等待下一毫秒
		if w.sequence == 0 {
			for {
				now2 := getUnixMillisecond()
				if now2 > now {
					w.lastMillisecond = now2
					break
				}
			}

		}
	}
	id := (w.lastMillisecond-beginMilliSecond)<<(workerIDBits+sequenceBits) | (w.workerID << sequenceBits) | w.sequence
	return id, nil
}

func Parse(id int64) (time int64, workerID int64, sequence int64, err error) {
	time = id >> (workerIDBits + sequenceBits)
	workerID = (id >> workerIDBits) & (time << sequenceBits)
	sequence = (id >> sequenceBits << sequenceBits) & id

	if workerID > maxWorkerID || sequence > maxSequence {
		err = invalidIDError
		return 0, 0, 0, err
	}
	return
}

// 获取毫秒!!! 1毫秒 = 10的6次方毫秒
func getUnixMillisecond() int64 {
	return time.Now().UnixNano() / 1000000
}
