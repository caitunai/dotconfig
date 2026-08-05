package cmd

import (
	"github.com/caitunai/dotconfig/queue"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

var subscriberID string

// queueCmd represents the queue command
var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "A command of queue listener to process jobs",
	Long:  "Start this command to process jobs in the queues.",
	Run: func(cmd *cobra.Command, _ []string) {
		err := queue.Init()
		if err != nil {
			log.Error().Err(err).Msg("init queue publisher failed with error")
			return
		}
		err = queue.Start(cmd.Context(), subscriberID)
		if err != nil {
			log.Error().Err(err).Msg("start queue processes failed with error")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(queueCmd)
	queueCmd.Flags().StringVarP(&subscriberID, "subscriber", "s", "", "The subscriber id")
}
