package queue

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/caitunai/dotconfig/queue/job"
	"github.com/spf13/viper"
)

var ErrPublishTopicMessage = errors.New("publish message to topic failed")

func Publish(j job.Job) error {
	topic := j.GetJobTopic()
	if topic == "" {
		topic = "default"
	}
	m := make(message.Metadata)
	m.Set("name", j.GetJobName())
	topicPrefix := viper.GetString("queue.prefix")
	err := publisher.Publish(topicPrefix+topic, &message.Message{
		UUID:     watermill.NewUUID(),
		Metadata: m,
		Payload:  j.GetJobData(),
	})
	return errors.Join(ErrPublishTopicMessage, err)
}
