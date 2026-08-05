package cmd

import (
	"errors"
	"fmt"
	"os"

	"ariga.io/atlas/atlasexec"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var atlasEnvName string

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tools using Atlas",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		// load the environment from .env file
		if err := godotenv.Load(); err != nil {
			log.Error().Err(err).Str("file", ".env").Msg("failed to load .env file")
		}
	},
}

var errAtlas = errors.New("atlas migrate error")

// init the atlas client
func getAtlasClient() (*atlasexec.Client, error) {
	workdir, err := os.Getwd()
	if err != nil {
		return nil, errors.Join(err, errAtlas)
	}

	client, err := atlasexec.NewClient(workdir, "atlas")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize atlas client: %w", err)
	}
	return client, nil
}

// inspectCmd equal atlas migrate inspect
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect the baseline of the database",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getAtlasClient()
		if err != nil {
			return err
		}

		params := &atlasexec.SchemaInspectParams{
			Env: atlasEnvName,
		}

		res, err := client.SchemaInspect(cmd.Context(), params)
		if err != nil {
			return fmt.Errorf("migration inspect failed: %w", err)
		}

		err = os.WriteFile("./atlas/schema/0-baseline.hcl", []byte(res), 0o644)
		if err != nil {
			return errors.Join(err, errAtlas)
		}
		log.Info().
			Str("schema", "./atlas/schema/0-baseline.hcl").
			Str("command", "inspect").
			Msg("inspecting schema successfully")
		return nil
	},
}

// makeCmd equal atlas migrate diff
var makeCmd = &cobra.Command{
	Use:   "make [migration_name]",
	Short: "Make migrations based on an existing database and state of changes",
	Example: `# Generate add_users_table migration file
migrate make add_users_table

# Generate add_users_table migration file for a specific environment
migrate make add_users_table --env dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("migration name is required")
		}
		client, err := getAtlasClient()
		if err != nil {
			return err
		}

		params := &atlasexec.MigrateDiffParams{
			Name: args[0],
			Env:  atlasEnvName,
		}

		_, err = client.MigrateDiff(cmd.Context(), params)
		normal := "The command returned more than one result, use Slice function instead"
		if err != nil && normal != err.Error() {
			return fmt.Errorf("create migration failed: %w", err)
		}

		log.Info().
			Str("command", "diff").
			Msg("create migration successfully")
		return nil
	},
}

// applyCmd equal atlas migrate apply
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Applies pending migrations to the database",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getAtlasClient()
		if err != nil {
			return err
		}

		params := &atlasexec.MigrateApplyParams{
			Env: atlasEnvName,
		}

		// support the baseline migration when initialed
		baseline, _ := cmd.Flags().GetString("baseline")
		if baseline != "" {
			params.BaselineVersion = baseline
		}

		res, err := client.MigrateApply(cmd.Context(), params)
		if err != nil {
			return fmt.Errorf("migration apply failed: %w", err)
		}

		log.Info().Str("command", "apply").Str("current_version", res.Target).Msg("Migration applied.")
		if len(res.Applied) > 0 {
			log.Info().Str("command", "apply").Int("files", len(res.Applied)).Msg("Migration applied successfully")
			for _, file := range res.Applied {
				log.Info().Str("command", "apply").Str("file", file.Name).Msg("File applied.")
			}
		} else {
			log.Info().Str("command", "apply").Msg("No migrations applied.")
		}

		return nil
	},
}

// statusCmd equal atlas migrate status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get migration status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getAtlasClient()
		if err != nil {
			return err
		}

		res, err := client.MigrateStatus(cmd.Context(), &atlasexec.MigrateStatusParams{
			Env: atlasEnvName,
		})
		if err != nil {
			return errors.Join(err, errAtlas)
		}

		log.Info().
			Str("environment", atlasEnvName).
			Str("status", res.Status).
			Int("migrations", len(res.Available)).
			Msg("migrations status")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.AddCommand(inspectCmd)
	migrateCmd.AddCommand(makeCmd)
	migrateCmd.AddCommand(applyCmd)
	migrateCmd.AddCommand(statusCmd)

	// The default environment is dev
	migrateCmd.PersistentFlags().StringVarP(&atlasEnvName, "env", "e", "dev", "Environment defined in atlas.hcl (local, dev, prod)")

	// The baseline parameter for apply command
	applyCmd.Flags().StringP("baseline", "b", "", "Baseline version for brownfield migration")
}
