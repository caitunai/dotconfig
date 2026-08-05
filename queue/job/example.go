package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"
)

type Example struct {
	Number int `json:"number"`
}

func (e *Example) GetJobName() string {
	return "example"
}

func (e *Example) GetJobTopic() string {
	return "default"
}

func (e *Example) GetJobData() message.Payload {
	data, err := json.Marshal(e)
	if err != nil {
		return []byte("")
	}
	return data
}

func (e *Example) ParseJob(data message.Payload) Job {
	n := &Example{}
	err := json.Unmarshal(data, n)
	if err != nil {
		log.Error().Err(err).Bytes("data", data).Msg("parse example job data failed")
	}
	return n
}

func (e *Example) RunJob(ctx context.Context) error {
	time.Sleep(time.Second)
	log.Ctx(ctx).Debug().Int("num", e.Number).Msg("run example job")
	return nil
}
