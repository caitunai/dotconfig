package cmd

import (
	"os"
	"strings"

	"github.com/caitunai/dotconfig/db"
	"github.com/caitunai/dotconfig/services/configload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "app",
	Short:             "A golang application",
	Long:              "A golang application server, with api and website ui",
	PersistentPreRunE: initializeRootConfiguration,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.EnableTraverseRunHooks = true
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $HOME/.app.toml)",
	)
}

func initializeRootConfiguration(cmd *cobra.Command, _ []string) error {
	// The generate command bootstraps the first keyring and therefore cannot
	// depend on an already readable encrypted configuration source.
	source := configload.Source(strings.ToLower(strings.TrimSpace(viper.GetString("configload.source"))))
	if cmd != generateConfigKeyCmd {
		if err := configureConfigEncryption(); err != nil {
			return err
		}
	}
	if !shouldLoadPublishedConfiguration(cmd, viper.GetBool("configload.enabled"), source) {
		return nil
	}
	if err := loadPublishedConfiguration(cmd.Context()); err != nil {
		return err
	}
	return nil
}

func shouldLoadPublishedConfiguration(cmd *cobra.Command, enabled bool, source configload.Source) bool {
	return enabled && (cmd != generateConfigKeyCmd || source != configload.SourceDatabase)
}

func initConfig() {
	if cfgFile != "" {
		// Use a config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		// Search config in home directory with name ".ai-canvas" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".app")
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		log.Trace().Msgf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Trace().Err(err).Send()
	}
	if viper.GetString("db.database") != "" {
		db.Conn()
	}
	initLogger()
}

func initLogger() {
	if viper.GetString("mode") != "dev" {
		switch viper.GetString("logger.level") {
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		}
	} else {
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr}
		log.Logger = log.Output(consoleWriter)
	}
	zerolog.DefaultContextLogger = &log.Logger
}
