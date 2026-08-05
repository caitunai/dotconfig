package cmd

import (
	"github.com/caitunai/dotconfig/api/server"
	"github.com/caitunai/dotconfig/cache"
	"github.com/caitunai/dotconfig/queue"
	"github.com/caitunai/dotconfig/redis"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Command to start api server",
	Long:  "Start the server, you should set the config file, named: .app.toml",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if err := validateConfigCenterSettings(); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, _ []string) {
		redis.Init()
		cache.InitCache()
		err := queue.Init()
		if err != nil {
			log.Error().
				Err(err).
				Str("method", "Run").
				Str("package", "cmd").
				Str("command", "serve").
				Msg("init the queue publisher failed")
		}
		s := server.NewServer(viper.GetString("port"), viper.GetString("mode"))
		s.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
