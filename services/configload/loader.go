// Package configload loads immutable published configuration into the
// application's Viper bootstrap process.
package configload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Source string

const (
	SourceDatabase Source = "database"
	SourceHTTP     Source = "http"
	maxHTTPBody           = 2 * 1024 * 1024
	maxHTTPTimeout        = time.Minute
)

var (
	ErrLoad               = errors.New("load published configuration failed")
	ErrInvalidSettings    = errors.New("invalid published configuration loader settings")
	ErrLoaderDisabled     = errors.New("published configuration loader is disabled")
	ErrNamespaceInvalid   = errors.New("configload.namespace must be a valid namespace identifier")
	ErrEnvironmentInvalid = errors.New("configload.environment must be a valid environment identifier")
	ErrUnsupportedSource  = errors.New("configload.source must be database or http")
	ErrHTTPBaseURLInvalid = errors.New("configload.http.baseURL must be an absolute HTTP or HTTPS URL without credentials, query, or fragment")
	ErrHTTPAPIKeyRequired = errors.New("configload.http.apiKey is required for the HTTP source")
	ErrHTTPTimeoutInvalid = errors.New("configload.http.timeout must be greater than zero and at most one minute")
	ErrDatabaseLoader     = errors.New("database configuration loader is unavailable")
	ErrHTTP               = errors.New("published configuration HTTP request failed")
	ErrInvalidResponse    = errors.New("invalid published configuration response")
	configSlugPattern     = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
)

type Settings struct {
	Source      Source
	Namespace   string
	Environment string
	HTTP        HTTPSettings
	Enabled     bool
}

type HTTPSettings struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type Result struct {
	Config      map[string]any
	Source      Source
	Namespace   string
	Environment string
	Version     uint64
}

type DatabaseLoader func(context.Context, string, string) (*Result, error)

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return "published configuration HTTP status " + strconv.Itoa(e.StatusCode)
}

func Load(ctx context.Context, settings Settings, databaseLoader DatabaseLoader) (*Result, error) {
	settings = normalizeSettings(settings)
	if err := validateSettings(settings); err != nil {
		return nil, errors.Join(ErrLoad, err)
	}
	var result *Result
	var err error
	switch settings.Source {
	case SourceDatabase:
		if databaseLoader == nil {
			return nil, errors.Join(ErrLoad, ErrInvalidSettings, ErrDatabaseLoader)
		}
		result, err = databaseLoader(ctx, settings.Namespace, settings.Environment)
	case SourceHTTP:
		result, err = loadHTTP(ctx, settings)
	default:
		return nil, errors.Join(ErrLoad, ErrUnsupportedSource)
	}
	if err != nil {
		return nil, errors.Join(ErrLoad, err)
	}
	if result == nil || result.Config == nil {
		return nil, errors.Join(ErrLoad, ErrInvalidResponse)
	}
	result.Source = settings.Source
	result.Namespace = settings.Namespace
	result.Environment = settings.Environment
	return result, nil
}

func normalizeSettings(settings Settings) Settings {
	settings.Source = Source(strings.ToLower(strings.TrimSpace(string(settings.Source))))
	settings.Namespace = strings.ToLower(strings.TrimSpace(settings.Namespace))
	settings.Environment = strings.ToLower(strings.TrimSpace(settings.Environment))
	settings.HTTP.BaseURL = strings.TrimRight(strings.TrimSpace(settings.HTTP.BaseURL), "/")
	settings.HTTP.APIKey = strings.TrimSpace(settings.HTTP.APIKey)
	return settings
}

func validateSettings(settings Settings) error {
	if !settings.Enabled {
		return errors.Join(ErrInvalidSettings, ErrLoaderDisabled)
	}
	if !configSlugPattern.MatchString(settings.Namespace) {
		return errors.Join(ErrInvalidSettings, ErrNamespaceInvalid)
	}
	if !configSlugPattern.MatchString(settings.Environment) {
		return errors.Join(ErrInvalidSettings, ErrEnvironmentInvalid)
	}
	switch settings.Source {
	case SourceDatabase:
		return nil
	case SourceHTTP:
		if settings.HTTP.APIKey == "" {
			return errors.Join(ErrInvalidSettings, ErrHTTPAPIKeyRequired)
		}
		if settings.HTTP.Timeout <= 0 || settings.HTTP.Timeout > maxHTTPTimeout {
			return errors.Join(ErrInvalidSettings, ErrHTTPTimeoutInvalid)
		}
		baseURL, err := url.Parse(settings.HTTP.BaseURL)
		if err != nil || baseURL.Host == "" || baseURL.User != nil ||
			(baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
			baseURL.RawQuery != "" || baseURL.Fragment != "" {
			return errors.Join(ErrInvalidSettings, ErrHTTPBaseURLInvalid)
		}
		return nil
	default:
		return ErrUnsupportedSource
	}
}

type httpEnvelope struct {
	Message string           `json:"message"`
	Data    httpResponseData `json:"data"`
	Status  int              `json:"status"`
}

type httpResponseData struct {
	Config  map[string]any `json:"config"`
	Version uint64         `json:"version"`
}

func loadHTTP(ctx context.Context, settings Settings) (*Result, error) {
	endpoint, err := url.JoinPath(
		settings.HTTP.BaseURL,
		"config-center/api/runtime",
		settings.Namespace,
		settings.Environment,
	)
	if err != nil {
		return nil, errors.Join(ErrHTTP, err)
	}
	envelope := &httpEnvelope{}
	client := resty.New().
		SetTimeout(settings.HTTP.Timeout).
		SetResponseBodyLimit(maxHTTPBody).
		SetRedirectPolicy(resty.NoRedirectPolicy()).
		SetCloseConnection(true).
		SetJSONUnmarshaler(unmarshalJSONNumber)
	response, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("X-API-Key", settings.HTTP.APIKey).
		SetQueryParam("format", "json").
		SetResult(envelope).
		Get(endpoint)
	if err != nil {
		return nil, errors.Join(ErrHTTP, err)
	}
	if !response.IsSuccess() {
		return nil, errors.Join(ErrHTTP, &HTTPStatusError{StatusCode: response.StatusCode()})
	}
	if envelope.Status != 0 || envelope.Data.Config == nil {
		return nil, errors.Join(ErrHTTP, ErrInvalidResponse)
	}
	return &Result{Config: envelope.Data.Config, Version: envelope.Data.Version}, nil
}

func unmarshalJSONNumber(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return errors.Join(ErrInvalidResponse, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return ErrInvalidResponse
		}
		return errors.Join(ErrInvalidResponse, err)
	}
	return nil
}
