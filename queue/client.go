package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/caitunai/dotconfig/queue/job"
	"github.com/caitunai/dotconfig/redis"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	publisher      *redisstream.Publisher
	subscriber     *redisstream.Subscriber
	jobs           = make(map[string]job.Job)
	ErrRedisStream = errors.New("create redis stream client failed")
)

func Init() error {
	var err error
	publisher, err = redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     redis.GetClient(),
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		NewLogger(),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", "Init").
			Str("package", "queue").
			Msg("create queue publisher failed")
		return errors.Join(ErrRedisStream, err)
	}
	return nil
}

func Start(ctx context.Context, subscriberID string) error {
	var err error
	subscriber, err = redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        redis.GetClient(),
			Consumer:      viper.GetString("queue.consumer.name") + subscriberID,
			ConsumerGroup: viper.GetString("queue.consumer.group"),
		},
		NewLogger(),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", "Init").
			Str("package", "queue").
			Msg("create queue subscriber failed")
		return errors.Join(ErrRedisStream, err)
	}
	SubscribeJob()
	// Wait for interrupt signal to gracefully shut down the queue with
	quit := make(chan os.Signal, 1)
	// kill (no param) default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	tCtx, cancel := context.WithCancel(ctx)
	subscribe(tCtx, quit, &wg)
	sig := <-quit
	cancel()
	wg.Wait()
	log.Info().Str("signal", sig.String()).Msg("exit to stop process queue jobs")
	return nil
}

func subscribe(ctx context.Context, kill chan os.Signal, wg *sync.WaitGroup) {
	topics := viper.GetStringSlice("queue.topics")
	if len(topics) == 0 {
		go listenTopic(ctx, "default", kill, wg)
		wg.Add(1)
	} else {
		for _, topic := range topics {
			go listenTopic(ctx, topic, kill, wg)
			wg.Add(1)
		}
	}
}

func listenTopic(ctx context.Context, topic string, kill chan os.Signal, wg *sync.WaitGroup) {
	defer func() {
		// catch panic
		if e := recover(); e != nil {
			log.Error().
				Str("topic", topic).
				Str("reason", fmt.Sprintf("recover: %v", e)).
				Bytes("stack", debug.Stack()).
				Msg("panic when listen topic")
			kill <- syscall.Signal(-10000)
		}
		wg.Done()
	}()
	topicPrefix := viper.GetString("queue.prefix")
	messages, err := subscriber.Subscribe(ctx, topicPrefix+topic)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", "listenTopic").
			Str("package", "queue").
			Msg("listen topic messages failed")
		return
	}
	log.Info().
		Str("topic", topic).
		Str("prefix", topicPrefix).
		Msg("subscribe topic successfully")
	for msg := range messages {
		if msg != nil {
			msg.Ack()
			dispatch(ctx, topic, msg.Metadata.Get("name"), msg.UUID, msg.Payload)
		}
	}
}

func dispatch(ctx context.Context, topic, name, id string, data message.Payload) {
	j, ok := jobs[name]
	if ok {
		err := j.ParseJob(data).RunJob(ctx)
		if err != nil {
			log.Error().
				Err(err).
				Str("topic", topic).
				Str("job_name", name).
				Bytes("data", data).
				Msg("job handler run failed")
		} else {
			log.Info().
				Str("topic", topic).
				Str("name", name).
				Str("job_id", id).
				Msg("run job successfully")
		}
	} else {
		log.Error().
			Str("topic", topic).
			Str("job_name", name).
			Bytes("data", data).
			Msg("job handler not found")
	}
}

func register(j job.Job) {
	jobs[j.GetJobName()] = j
}
