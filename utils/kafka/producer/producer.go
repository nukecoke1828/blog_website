package producer

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{}, // 负载均衡策略
			RequiredAcks: kafka.RequireAll,    // 等待所有副本确认
			BatchSize:    100,                 // 批量发送，可根据场景调整
			BatchTimeout: 10 * time.Millisecond,
			Async:        true,
			Completion: func(messages []kafka.Message, err error) { // 异步发送完成回调
				if err != nil {
					log.Printf("kafka async send failed: %v", err)
				}
			},
		},
	}
}

func (p *KafkaProducer) SendComment(ctx context.Context, comment *models.Comment) error {
	payload, err := json.Marshal(comment)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(strconv.FormatUint(uint64(comment.BlogID), 10)), // 相同 Key 进入同一分区，保证顺序性
		Value: payload,
		Time:  time.Now(),
	}
	// 异步写入，不阻塞主流程
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("kafka send failed: %v", err)
		return err
	}
	log.Printf("comment sent to kafka: %s", []byte(strconv.FormatUint(uint64(comment.BlogID), 10)))
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
