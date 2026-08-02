package kafka

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/nukecoke1828/my_blog_website/utils/kafka/consumer"
	"github.com/nukecoke1828/my_blog_website/utils/kafka/producer"
	"gorm.io/gorm"
)

var (
	cons   *consumer.BatchConsumer
	prod   *producer.KafkaProducer
	cancel context.CancelFunc
	once   sync.Once
)

func Init(db *gorm.DB) {
	once.Do(func() {
		// 从环境变量读取，默认 localhost:9092（本地开发兼容）
		brokers := []string{"localhost:9092"}
		if env := os.Getenv("KAFKA_BROKERS"); env != "" {
			brokers = strings.Split(env, ",") // 支持多节点，逗号分隔
		}

		topic := "comment_topic"
		groupID := "comment_consumer_group"

		ctx, c := context.WithCancel(context.Background())
		cancel = c

		cons = consumer.NewBatchConsumer(brokers, topic, groupID, db)
		cons.Start(ctx)

		prod = producer.NewKafkaProducer(brokers, topic)
	})
}

func Producer() *producer.KafkaProducer {
	return prod
}

// Shutdown 直接释放资源，不等待信号
func Shutdown() {
	if cancel != nil {
		cancel()
	}
	if cons != nil {
		cons.Stop()
	}
	if prod != nil {
		prod.Close()
	}
}
