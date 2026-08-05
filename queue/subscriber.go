package queue

import "github.com/caitunai/dotconfig/queue/job"

func SubscribeJob() {
	register(&job.Example{})
}
