package job

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type Job interface {
	GetJobName() string
	GetJobTopic() string
	GetJobData() message.Payload
	ParseJob(data message.Payload) Job
	RunJob(ctx context.Context) error
}
