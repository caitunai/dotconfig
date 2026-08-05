package cmd

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/caitunai/dotconfig/db"
	"github.com/caitunai/dotconfig/services/configload"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const defaultConfigLoadHTTPTimeout = 5 * time.Second

var (
	ErrPublishedConfigLoad  = errors.New("published configuration bootstrap failed")
	ErrPublishedConfigMerge = errors.New("merge published configuration into Viper failed")
	ErrPublishedConfigDB    = errors.New("load published configuration from database failed")
	protectedBootstrapRoots = map[string]struct{}{
		"configcenter": {},
		"configcrypt":  {},
		"configload":   {},
		"db":           {},
	}
)

func loadPublishedConfiguration(ctx context.Context) error {
	if !viper.GetBool("configload.enabled") {
		return nil
	}
	timeout := viper.GetDuration("configload.http.timeout")
	if timeout == 0 {
		timeout = defaultConfigLoadHTTPTimeout
	}
	settings := configload.Settings{
		Enabled:     true,
		Source:      configload.Source(viper.GetString("configload.source")),
		Namespace:   viper.GetString("configload.namespace"),
		Environment: viper.GetString("configload.environment"),
		HTTP: configload.HTTPSettings{
			BaseURL: viper.GetString("configload.http.baseURL"),
			APIKey:  viper.GetString("configload.http.apiKey"),
			Timeout: timeout,
		},
	}
	result, err := configload.Load(ctx, settings, loadPublishedConfigurationFromDatabase)
	if err != nil {
		return errors.Join(ErrPublishedConfigLoad, err)
	}
	if err := mergePublishedConfiguration(viper.GetViper(), result.Config); err != nil {
		return errors.Join(ErrPublishedConfigLoad, err)
	}
	log.Info().
		Str("source", string(result.Source)).
		Str("namespace", result.Namespace).
		Str("environment", result.Environment).
		Uint64("version", result.Version).
		Msg("published configuration loaded")
	return nil
}

func loadPublishedConfigurationFromDatabase(
	ctx context.Context,
	namespaceSlug string,
	environmentSlug string,
) (*configload.Result, error) {
	_, _, published, err := db.LatestPublishedConfigBySlugsInternal(ctx, namespaceSlug, environmentSlug)
	if err != nil {
		return nil, errors.Join(ErrPublishedConfigDB, err)
	}
	return &configload.Result{
		Config:  published.Config,
		Version: published.Version,
	}, nil
}

func mergePublishedConfiguration(target *viper.Viper, config map[string]any) error {
	filtered := make(map[string]any, len(config))
	for key, value := range config {
		if _, protected := protectedBootstrapRoots[strings.ToLower(key)]; protected {
			continue
		}
		filtered[key] = clonePublishedConfigValue(value)
	}
	if err := target.MergeConfigMap(filtered); err != nil {
		return errors.Join(ErrPublishedConfigMerge, err)
	}
	return nil
}

func clonePublishedConfigValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, child := range typed {
			cloned[key] = clonePublishedConfigValue(child)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, child := range typed {
			cloned[index] = clonePublishedConfigValue(child)
		}
		return cloned
	default:
		return typed
	}
}
