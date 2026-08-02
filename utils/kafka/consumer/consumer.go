package consumer

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	db     *gorm.DB
	wg     sync.WaitGroup
	stop   chan struct{}
}

// BatchConsumer 批量消费者
type BatchConsumer struct {
	reader *kafka.Reader
	db     *gorm.DB
	batch  []kafka.Message
	size   int           // 批量大小
	flush  time.Duration // 刷新间隔
	stop   chan struct{}
	wg     sync.WaitGroup
}

func NewKafkaConsumer(brokers []string, topic, groupID string, db *gorm.DB) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,          // 消费者组，实现负载均衡和故障转移
			MinBytes:       10e3,             // 10KB
			MaxBytes:       10e6,             // 10MB
			MaxWait:        time.Second,      // 最长等待时间
			CommitInterval: time.Second * 5,  // 自动提交偏移量间隔
			StartOffset:    kafka.LastOffset, // 从最新消息开始消费
		}),
		db:   db,
		stop: make(chan struct{}),
	}
}

func NewBatchConsumer(brokers []string, topic, groupID string, db *gorm.DB) *BatchConsumer {
	return &BatchConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			MinBytes:    10e3,
			MaxBytes:    10e6,
			MaxWait:     time.Second,
			StartOffset: kafka.LastOffset,
			// 注意：批量消费建议手动提交，这里关闭自动提交
			CommitInterval: 0,
		}),
		db:    db,
		size:  100,         // 每 100 条刷一次盘
		flush: time.Second, // 最多等 1 秒
		stop:  make(chan struct{}),
	}
}

// Start 启动消费
func (c *KafkaConsumer) Start(ctx context.Context) {
	c.wg.Add(1)
	go c.run(ctx)
}

func (c *BatchConsumer) Start(ctx context.Context) {
	c.wg.Add(1)
	go c.run(ctx)
}

func (c *KafkaConsumer) run(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stop:
			log.Println("consumer stopping...")
			return
		default:
		}

		// 拉取消息，设置超时避免永久阻塞
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Printf("fetch message error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// 处理单条消息（也可改为批量处理）
		if err := c.handleMessage(ctx, msg); err != nil {
			log.Printf("handle message failed: %v", err)
			// 这里可以加入重试逻辑或死信队列
			continue
		}

		// 手动提交偏移量（如果上面 CommitInterval 设为 0，则需要手动 commit）
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("commit failed: %v", err)
		}
	}
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	var comment models.Comment
	if err := json.Unmarshal(msg.Value, &comment); err != nil {
		log.Printf("unmarshal failed: %v, raw: %s", err, string(msg.Value))
		return nil // 格式错误直接丢弃，避免阻塞
	}

	// 写入数据库，使用 msg_id 做幂等：重复消息自动跳过
	result := c.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "msg_id"}},
		DoNothing: true,
	}).Create(&comment)
	if result.Error != nil {
		return result.Error
	}

	log.Printf("comment saved to db: blog_id=%d, partition: %d, offset: %d",
		comment.BlogID, msg.Partition, msg.Offset)
	return nil
}

func (c *KafkaConsumer) Stop() {
	close(c.stop)
	c.reader.Close()
	c.wg.Wait()
}

func (c *BatchConsumer) Stop() {
	close(c.stop)
	c.reader.Close()
	c.wg.Wait()
}

func (c *BatchConsumer) run(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flush)
	defer ticker.Stop()

	type result struct {
		msg kafka.Message
		err error
	}
	fetchChan := make(chan result, 1)

	// 独立 goroutine 阻塞拉取消息，不再被超时打断
	go func() {
		for {
			msg, err := c.reader.FetchMessage(ctx)
			select {
			case fetchChan <- result{msg: msg, err: err}:
			case <-ctx.Done():
				return
			case <-c.stop:
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			c.flushBatch(context.Background())
			return
		case <-c.stop:
			c.flushBatch(context.Background())
			return
		case <-ticker.C:
			c.flushBatch(ctx)
		case res := <-fetchChan:
			if res.err != nil {
				log.Printf("fetch error: %v", res.err)
				time.Sleep(time.Second)
				continue
			}
			c.batch = append(c.batch, res.msg)
			if len(c.batch) >= c.size {
				c.flushBatch(ctx)
			}
		}
	}
}

func (c *BatchConsumer) flushBatch(ctx context.Context) {
	if len(c.batch) == 0 {
		return
	}

	// 真正复制切片，避免和 c.batch 共享底层数组
	batch := make([]kafka.Message, len(c.batch))
	copy(batch, c.batch)
	c.batch = c.batch[:0] // 立即清空，新消息可以继续进入

	var comments []models.Comment
	var commitBatch []kafka.Message

	for _, msg := range batch {
		var comment models.Comment
		if err := json.Unmarshal(msg.Value, &comment); err != nil {
			log.Printf("unmarshal failed: %v, raw: %s", err, string(msg.Value))
			commitBatch = append(commitBatch, msg)
			continue
		}
		comments = append(comments, comment)
		commitBatch = append(commitBatch, msg)
	}

	if len(comments) > 0 {
		err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "msg_id"}},
				DoNothing: true,
			}).CreateInBatches(comments, len(comments)).Error
		})
		if err != nil {
			log.Printf("batch insert failed: %v", err)
			// 失败时把消息加回 batch 头部，下次重试
			c.batch = append(batch, c.batch...)
			// 失败时直接退出，不再提交偏移量，下次重试
			return
		}
	}

	if len(commitBatch) > 0 {
		if err := c.reader.CommitMessages(ctx, commitBatch...); err != nil {
			log.Printf("batch commit failed: %v", err)
			c.batch = append(batch, c.batch...)
			return
		}
	}

	log.Printf("flushed %d messages", len(commitBatch))
}
