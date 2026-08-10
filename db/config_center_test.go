package db

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/caitunai/dotconfig/services/configcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testBase               = "base"
	testFlags              = "flags"
	testHost               = "host"
	testPort               = "port"
	testProd               = "prod"
	testProduction         = "production"
	testRegions            = "regions"
	testService            = "service"
	testEnabled            = "enabled"
	testName               = "name"
	testRegionsPointer     = "/regions"
	testServicePointer     = "/service"
	testServiceHostPointer = "/service/host"
	testServiceSettings    = "service settings"
)

func TestNamespaceAPIKeyIsEncryptedAndAuthenticated(t *testing.T) {
	keyringPath := configureTestConfigEncryption(t)

	readAPIKey := strings.Repeat("a", minConfigAPIKeyLength)
	writeAPIKey := strings.Repeat("b", minConfigAPIKeyLength)
	storedReadKey, err := encryptNamespaceAPIKey(readAPIKey, 7)
	if err != nil {
		t.Fatalf("encryptNamespaceAPIKey() error = %v", err)
	}
	storedWriteKey, err := encryptNamespaceWriteAPIKey(writeAPIKey, 7)
	if err != nil {
		t.Fatalf("encryptNamespaceWriteAPIKey() error = %v", err)
	}
	if storedReadKey == readAPIKey || strings.Contains(storedReadKey, readAPIKey) ||
		storedWriteKey == writeAPIKey || strings.Contains(storedWriteKey, writeAPIKey) {
		t.Fatal("encryptNamespaceAPIKey() stored plaintext API key")
	}
	namespace := &ConfigNamespace{ID: 7, APIKey: storedReadKey, WriteAPIKey: storedWriteKey}
	publicView, err := json.Marshal(namespace)
	if err != nil {
		t.Fatalf("Marshal() namespace error = %v", err)
	}
	if strings.Contains(string(publicView), storedReadKey) || strings.Contains(string(publicView), readAPIKey) ||
		strings.Contains(string(publicView), storedWriteKey) || strings.Contains(string(publicView), writeAPIKey) {
		t.Fatal("ConfigNamespace JSON exposed API key material")
	}
	if err := authenticateNamespaceAPIKey(namespace, readAPIKey); err != nil {
		t.Fatalf("authenticateNamespaceAPIKey() error = %v", err)
	}
	if err := authenticateNamespaceWriteAPIKey(namespace, writeAPIKey); err != nil {
		t.Fatalf("authenticateNamespaceWriteAPIKey() error = %v", err)
	}
	if err := authenticateNamespaceAPIKey(namespace, writeAPIKey); !errors.Is(err, ErrConfigAPIKeyUnauthorized) {
		t.Fatalf("authenticateNamespaceAPIKey(write key) error = %v, want ErrConfigAPIKeyUnauthorized", err)
	}
	if err := authenticateNamespaceWriteAPIKey(namespace, readAPIKey); !errors.Is(err, ErrConfigWriteAPIKeyUnauthorized) {
		t.Fatalf("authenticateNamespaceWriteAPIKey(read key) error = %v, want ErrConfigWriteAPIKeyUnauthorized", err)
	}
	if err := configcrypt.GenerateFileKey(keyringPath, "namespace-test-key-v2"); err != nil {
		t.Fatalf("GenerateFileKey(v2) error = %v", err)
	}
	if err := configcrypt.Configure(configcrypt.Settings{
		Enabled:     true,
		Provider:    configcrypt.ProviderFile,
		ActiveKeyID: "namespace-test-key-v2",
		KeyringPath: keyringPath,
	}); err != nil {
		t.Fatalf("Configure(v2) error = %v", err)
	}
	changes, payloads, err := reencryptNamespaceCredentials(namespace)
	if err != nil {
		t.Fatalf("reencryptNamespaceCredentials() error = %v", err)
	}
	if len(changes) != 2 || payloads != 2 {
		t.Fatalf("reencryptNamespaceCredentials() changes = %#v, payloads = %d; want two", changes, payloads)
	}
	namespace.APIKey = changes[columnAPIKey].(string)
	namespace.WriteAPIKey = changes[columnWriteAPIKey].(string)
	if err := authenticateNamespaceAPIKey(namespace, readAPIKey); err != nil {
		t.Fatalf("authenticateNamespaceAPIKey(reencrypted) error = %v", err)
	}
	if err := authenticateNamespaceWriteAPIKey(namespace, writeAPIKey); err != nil {
		t.Fatalf("authenticateNamespaceWriteAPIKey(reencrypted) error = %v", err)
	}
}

func configureTestConfigEncryption(t *testing.T) string {
	t.Helper()
	keyringPath := filepath.Join(t.TempDir(), "keys.json")
	if err := configcrypt.GenerateFileKey(keyringPath, "namespace-test-key"); err != nil {
		t.Fatalf("GenerateFileKey() error = %v", err)
	}
	if err := configcrypt.Configure(configcrypt.Settings{
		Enabled:     true,
		Provider:    configcrypt.ProviderFile,
		ActiveKeyID: "namespace-test-key",
		KeyringPath: keyringPath,
	}); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	t.Cleanup(func() {
		if err := configcrypt.Configure(configcrypt.Settings{}); err != nil {
			t.Fatalf("reset Configure() error = %v", err)
		}
	})
	return keyringPath
}

func TestUpdatePublishedConfigValueBySlugsPublishesInheritedOverride(t *testing.T) {
	_ = configureTestConfigEncryption(t)
	previousDB := db
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "config-center.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	db = connection
	t.Cleanup(func() { db = previousDB })
	if err := connection.AutoMigrate(&ConfigNamespace{}, &ConfigEnvironment{}, &ConfigRelease{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	readAPIKey := strings.Repeat("r", minConfigAPIKeyLength)
	writeAPIKey := strings.Repeat("w", minConfigAPIKeyLength)
	namespace, err := CreateConfigNamespace(t.Context(), ConfigNamespaceInput{
		Name:        "Service",
		Slug:        testService,
		APIKey:      readAPIKey,
		WriteAPIKey: writeAPIKey,
	})
	if err != nil {
		t.Fatalf("CreateConfigNamespace() error = %v", err)
	}
	if _, err := UpdateConfigNamespace(t.Context(), namespace.ID, ConfigNamespaceInput{
		Name:        namespace.Name,
		Slug:        namespace.Slug,
		WriteAPIKey: readAPIKey,
	}); !errors.Is(err, ErrConfigAPIKeysNotDistinct) {
		t.Fatalf("UpdateConfigNamespace(shared key) error = %v, want ErrConfigAPIKeysNotDistinct", err)
	}
	baseEnvironment, err := CreateConfigEnvironment(t.Context(), namespace.ID, ConfigEnvironmentInput{
		Name: testBase,
		Slug: testBase,
	})
	if err != nil {
		t.Fatalf("CreateConfigEnvironment(base) error = %v", err)
	}
	production, err := CreateConfigEnvironment(t.Context(), namespace.ID, ConfigEnvironmentInput{
		Name:     testProd,
		Slug:     testProd,
		ParentID: baseEnvironment.ID,
	})
	if err != nil {
		t.Fatalf("CreateConfigEnvironment(prod) error = %v", err)
	}
	if _, err := SaveConfigDraft(
		t.Context(),
		namespace.ID,
		baseEnvironment.ID,
		[]byte(`{"service":{"host":"base","port":80}}`),
		[]byte(`{}`),
	); err != nil {
		t.Fatalf("SaveConfigDraft() error = %v", err)
	}
	if _, err := PublishConfigs(t.Context(), namespace.ID, []uint{production.ID}); err != nil {
		t.Fatalf("PublishConfigs() error = %v", err)
	}
	if _, _, _, err := UpdatePublishedConfigValueBySlugs(
		t.Context(),
		namespace.Slug,
		production.Slug,
		readAPIKey,
		testServiceHostPointer,
		[]byte(`"denied"`),
	); !errors.Is(err, ErrConfigWriteAPIKeyUnauthorized) {
		t.Fatalf("UpdatePublishedConfigValueBySlugs(read key) error = %v, want ErrConfigWriteAPIKeyUnauthorized", err)
	}

	_, _, published, err := UpdatePublishedConfigValueBySlugs(
		t.Context(),
		namespace.Slug,
		production.Slug,
		writeAPIKey,
		testServiceHostPointer,
		[]byte(`"`+testProduction+`"`),
	)
	if err != nil {
		t.Fatalf("UpdatePublishedConfigValueBySlugs() error = %v", err)
	}
	if published.Version != 2 || published.Config[testService].(map[string]any)[testHost] != testProduction {
		t.Fatalf("published = %#v, want version 2 with production host", published)
	}
	if published.Config[testService].(map[string]any)[testPort] != json.Number("80") {
		t.Fatal("published configuration lost an inherited sibling value")
	}

	_, draft, _, err := GetConfigDraft(t.Context(), namespace.ID, production.ID)
	if err != nil {
		t.Fatalf("GetConfigDraft() error = %v", err)
	}
	service, ok := draft[testService].(map[string]any)
	if !ok || len(service) != 1 || service[testHost] != testProduction {
		t.Fatalf("draft = %#v, want minimal host-only override", draft)
	}
	_, _, latest, err := LatestPublishedConfigBySlugs(t.Context(), namespace.Slug, production.Slug, readAPIKey)
	if err != nil {
		t.Fatalf("LatestPublishedConfigBySlugs() error = %v", err)
	}
	if latest.Version != published.Version || latest.Config[testService].(map[string]any)[testHost] != testProduction {
		t.Fatalf("latest = %#v, want newly published configuration", latest)
	}
	if _, _, _, err := LatestPublishedConfigBySlugs(t.Context(), namespace.Slug, production.Slug, writeAPIKey); !errors.Is(err, ErrConfigAPIKeyUnauthorized) {
		t.Fatalf("LatestPublishedConfigBySlugs(write key) error = %v, want ErrConfigAPIKeyUnauthorized", err)
	}
}

func TestValidateNamespaceAPIKey(t *testing.T) {
	t.Parallel()
	base := ConfigNamespaceInput{Name: "Service", Slug: testService}
	if err := validateConfigNamespaceInput(base, true); !errors.Is(err, ErrConfigAPIKeyInvalid) {
		t.Fatalf("validateConfigNamespaceInput() missing key error = %v", err)
	}
	base.APIKey = strings.Repeat("a", minConfigAPIKeyLength)
	if err := validateConfigNamespaceInput(base, true); !errors.Is(err, ErrConfigWriteAPIKeyInvalid) {
		t.Fatalf("validateConfigNamespaceInput() missing write key error = %v", err)
	}
	base.WriteAPIKey = strings.Repeat("b", minConfigAPIKeyLength)
	if err := validateConfigNamespaceInput(base, true); err != nil {
		t.Fatalf("validateConfigNamespaceInput() error = %v", err)
	}
	base.WriteAPIKey = base.APIKey
	if err := validateConfigNamespaceInput(base, true); !errors.Is(err, ErrConfigAPIKeysNotDistinct) {
		t.Fatalf("validateConfigNamespaceInput() shared key error = %v, want ErrConfigAPIKeysNotDistinct", err)
	}
	base.WriteAPIKey = strings.Repeat("b", minConfigAPIKeyLength)
	base.APIKey = strings.Repeat("a", minConfigAPIKeyLength-1)
	if err := validateConfigNamespaceInput(base, false); !errors.Is(err, ErrConfigAPIKeyInvalid) {
		t.Fatalf("validateConfigNamespaceInput() short key error = %v", err)
	}
	base.APIKey = strings.Repeat("a", minConfigAPIKeyLength)
	base.WriteAPIKey = strings.Repeat("b", minConfigAPIKeyLength-1)
	if err := validateConfigNamespaceInput(base, false); !errors.Is(err, ErrConfigWriteAPIKeyInvalid) {
		t.Fatalf("validateConfigNamespaceInput() short write key error = %v", err)
	}
}

func TestDecodeConfigPreservesSupportedTypes(t *testing.T) {
	t.Parallel()

	config, err := DecodeConfig([]byte(`{
		"object":{"enabled":true},
		"array":["a",2,false],
		"string":"value",
		"bool":true,
		"int":42,
		"float":3.14
	}`))
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if got := config["int"]; got != json.Number("42") {
		t.Fatalf("int = %#v, want json.Number(42)", got)
	}
	if got := config["float"]; got != json.Number("3.14") {
		t.Fatalf("float = %#v, want json.Number(3.14)", got)
	}
}

func TestDecodeConfigRejectsUnsupportedValues(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"null":           `{"value":null}`,
		"non-object":     `[1,2,3]`,
		"empty key":      `{" ":true}`,
		"multiple roots": `{} {}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeConfig([]byte(raw))
			if !errors.Is(err, ErrConfigInvalid) {
				t.Fatalf("DecodeConfig() error = %v, want ErrConfigInvalid", err)
			}
		})
	}
}

func TestDecodeConfigRejectsExcessiveDepth(t *testing.T) {
	t.Parallel()

	raw := strings.Repeat(`{"nested":`, maxConfigDepth+1) + `true` + strings.Repeat("}", maxConfigDepth+1)
	_, err := DecodeConfig([]byte(raw))
	if !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("DecodeConfig() error = %v, want ErrConfigInvalid", err)
	}
}

func TestDecodeConfigValuePreservesTypesAndRejectsNull(t *testing.T) {
	t.Parallel()

	value, err := decodeConfigValue([]byte(`{"` + testEnabled + `":true,"retries":3}`))
	if err != nil {
		t.Fatalf("decodeConfigValue() error = %v", err)
	}
	object, ok := value.(map[string]any)
	if !ok || object[testEnabled] != true || object["retries"] != json.Number("3") {
		t.Fatalf("decodeConfigValue() = %#v, want typed object", value)
	}
	if _, err := decodeConfigValue([]byte(`null`)); !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("decodeConfigValue(null) error = %v, want ErrConfigInvalid", err)
	}
	if _, err := decodeConfigValue([]byte(`true false`)); !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("decodeConfigValue(multiple) error = %v, want ErrConfigInvalid", err)
	}
}

func TestSetConfigOverrideValueWritesMinimalNestedOverride(t *testing.T) {
	t.Parallel()

	resolved := map[string]any{
		testService: map[string]any{testHost: testBase, testPort: json.Number("80")},
		testFlags:   map[string]any{testEnabled: true},
	}
	draft := map[string]any{
		testService: map[string]any{testPort: json.Number("8080")},
	}
	path, err := parseJSONPointer(testServiceHostPointer)
	if err != nil {
		t.Fatalf("parseJSONPointer() error = %v", err)
	}
	if err := setConfigOverrideValue(draft, resolved, path, testProd); err != nil {
		t.Fatalf("setConfigOverrideValue() error = %v", err)
	}
	service := draft[testService].(map[string]any)
	if service[testHost] != testProd || service[testPort] != json.Number("8080") {
		t.Fatalf("draft service = %#v, want existing override plus updated host", service)
	}
	if _, exists := draft[testFlags]; exists {
		t.Fatal("setConfigOverrideValue() copied unrelated inherited fields")
	}
}

func TestSetConfigOverrideValueCopiesResolvedArrayBeforeUpdatingItem(t *testing.T) {
	t.Parallel()

	resolved := map[string]any{
		testRegions: []any{
			map[string]any{testName: "cn", testEnabled: true},
			map[string]any{testName: "us", testEnabled: true},
		},
	}
	draft := make(map[string]any)
	path, err := parseJSONPointer("/regions/1/enabled")
	if err != nil {
		t.Fatalf("parseJSONPointer() error = %v", err)
	}
	if err := setConfigOverrideValue(draft, resolved, path, false); err != nil {
		t.Fatalf("setConfigOverrideValue() error = %v", err)
	}
	regions := draft[testRegions].([]any)
	if regions[0].(map[string]any)[testName] != "cn" || regions[1].(map[string]any)[testEnabled] != false {
		t.Fatalf("draft regions = %#v, want full array with one updated value", regions)
	}
	if resolved[testRegions].([]any)[1].(map[string]any)[testEnabled] != true {
		t.Fatal("setConfigOverrideValue() mutated the resolved configuration")
	}
	arrayPointer := configArrayOverridePointer(resolved, path)
	if arrayPointer != testRegionsPointer {
		t.Fatalf("configArrayOverridePointer() = %q, want /regions", arrayPointer)
	}
	descriptions := make(ConfigDescriptions)
	copyDescriptionSubtree(descriptions, ConfigDescriptions{
		testRegionsPointer:   "deployment regions",
		"/regions/0/name":    "primary region",
		"/regions/1/enabled": "secondary region switch",
	}, arrayPointer)
	removeDescriptionDescendants(descriptions, "/regions/1/enabled")
	if len(descriptions) != 3 || descriptions["/regions/0/name"] != "primary region" {
		t.Fatalf("materialized array descriptions = %#v, want inherited description subtree", descriptions)
	}
	if pointer := configArrayOverridePointer(resolved, []string{testRegions}); pointer != "" {
		t.Fatalf("configArrayOverridePointer(direct replacement) = %q, want empty", pointer)
	}
}

func TestSetConfigOverrideValueSupportsEscapedKeysAndRejectsMissingPath(t *testing.T) {
	t.Parallel()

	resolved := map[string]any{"service/api": map[string]any{"token~name": "old"}}
	draft := make(map[string]any)
	path, err := parseJSONPointer("/service~1api/token~0name")
	if err != nil {
		t.Fatalf("parseJSONPointer() error = %v", err)
	}
	if err := setConfigOverrideValue(draft, resolved, path, "new"); err != nil {
		t.Fatalf("setConfigOverrideValue() error = %v", err)
	}
	if got := draft["service/api"].(map[string]any)["token~name"]; got != "new" {
		t.Fatalf("escaped key value = %#v, want new", got)
	}
	if err := setConfigOverrideValue(draft, resolved, []string{"missing"}, true); !errors.Is(err, ErrConfigPathNotFound) {
		t.Fatalf("setConfigOverrideValue(missing) error = %v, want ErrConfigPathNotFound", err)
	}
}

func TestRemoveDescriptionDescendantsKeepsUpdatedPathDescription(t *testing.T) {
	t.Parallel()

	descriptions := ConfigDescriptions{
		testServicePointer:     testServiceSettings,
		testServiceHostPointer: "old host",
		"/other":               "unrelated",
	}
	removeDescriptionDescendants(descriptions, testServicePointer)
	if descriptions[testServicePointer] != testServiceSettings || descriptions["/other"] != "unrelated" {
		t.Fatalf("descriptions = %#v, want exact and unrelated descriptions", descriptions)
	}
	if _, exists := descriptions[testServiceHostPointer]; exists {
		t.Fatal("removeDescriptionDescendants() kept a stale child description")
	}
}

func TestMergeConfigRecursivelyMergesObjectsAndReplacesArrays(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		testService: map[string]any{testHost: testBase, testPort: json.Number("80")},
		testFlags:   []any{testBase},
		"keep":      true,
	}
	override := map[string]any{
		testService: map[string]any{testPort: json.Number("443")},
		testFlags:   []any{"child", "replacement"},
	}
	want := map[string]any{
		testService: map[string]any{testHost: testBase, testPort: json.Number("443")},
		testFlags:   []any{"child", "replacement"},
		"keep":      true,
	}
	got := mergeConfig(base, override)
	gotJSON, gotErr := json.Marshal(got)
	wantJSON, wantErr := json.Marshal(want)
	if gotErr != nil || wantErr != nil || string(gotJSON) != string(wantJSON) {
		t.Fatalf("mergeConfig() = %#v, want %#v", got, want)
	}
	got.(map[string]any)[testService].(map[string]any)[testHost] = "changed"
	if base[testService].(map[string]any)[testHost] != testBase {
		t.Fatal("mergeConfig() mutated the base configuration")
	}
}

func TestDecodeConfigDescriptionsValidatesJSONPointers(t *testing.T) {
	t.Parallel()

	config, err := DecodeConfig([]byte(`{"service/api":{"host":"localhost"},"regions":["cn"]}`))
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	descriptions, err := DecodeConfigDescriptions(
		[]byte(`{"/service~1api":" service settings ","/service~1api/host":"host","/regions/0":"China"}`),
		config,
	)
	if err != nil {
		t.Fatalf("DecodeConfigDescriptions() error = %v", err)
	}
	if descriptions["/service~1api"] != "service settings" {
		t.Fatalf("description = %q, want normalized value", descriptions["/service~1api"])
	}

	_, err = DecodeConfigDescriptions([]byte(`{"/regions/1":"missing"}`), config)
	if !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("DecodeConfigDescriptions() error = %v, want ErrConfigInvalid", err)
	}
}

func TestMergeConfigWithDescriptionsReplacesDescriptionSubtrees(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		testService: map[string]any{testHost: testBase, testPort: json.Number("80")},
		testRegions: []any{"cn", "us"},
	}
	override := map[string]any{
		testService: map[string]any{testHost: testProd},
		testRegions: []any{"cn"},
	}
	merged, descriptions := mergeConfigWithDescriptions(
		base,
		ConfigDescriptions{
			"/service":             "service",
			testServiceHostPointer: "base host",
			"/regions/1":           "US",
		},
		override,
		ConfigDescriptions{
			testServiceHostPointer: "production host",
			"/regions":             testRegions,
			"/regions/0":           "China",
		},
		"",
	)
	want := ConfigDescriptions{
		"/service":             "service",
		testServiceHostPointer: "production host",
		"/regions":             testRegions,
		"/regions/0":           "China",
	}
	if !equalDescriptions(descriptions, want) {
		t.Fatalf("descriptions = %#v, want %#v", descriptions, want)
	}
	if merged.(map[string]any)[testService].(map[string]any)[testPort] != json.Number("80") {
		t.Fatal("mergeConfigWithDescriptions() did not preserve inherited object fields")
	}
}

func equalDescriptions(left, right ConfigDescriptions) bool {
	if len(left) != len(right) {
		return false
	}
	for pointer, description := range left {
		if right[pointer] != description {
			return false
		}
	}
	return true
}

func TestResolveConfigFromEnvironmentsBuildsRootToLeafChain(t *testing.T) {
	t.Parallel()

	environments := []ConfigEnvironment{
		{ID: 3, ParentID: 2, Name: testProd, DraftConfig: `{"service":{"host":"prod"}}`},
		{ID: 1, Name: testBase, DraftConfig: `{"service":{"host":"base","port":80},"enabled":true}`},
		{ID: 2, ParentID: 1, Name: "region", DraftConfig: `{"service":{"port":8080}}`},
	}
	resolved, err := resolveConfigFromEnvironments(environments, 3)
	if err != nil {
		t.Fatalf("resolveConfigFromEnvironments() error = %v", err)
	}
	if got := []uint{resolved.Chain[0].ID, resolved.Chain[1].ID, resolved.Chain[2].ID}; !slices.Equal(got, []uint{1, 2, 3}) {
		t.Fatalf("chain IDs = %v, want [1 2 3]", got)
	}
	service := resolved.Config[testService].(map[string]any)
	if service[testHost] != testProd || service[testPort] != json.Number("8080") {
		t.Fatalf("resolved service = %#v", service)
	}
}

func TestApplyConfigEnvironmentReleaseStateDetectsDraftChanges(t *testing.T) {
	t.Parallel()

	environments := []ConfigEnvironment{
		{ID: 1, DraftConfig: `{"service":{"host":"base"}}`, DraftDescriptions: `{}`},
		{ID: 2, ParentID: 1, DraftConfig: `{"service":{"port":8080}}`, DraftDescriptions: `{}`},
		{ID: 3, DraftConfig: `{}`, DraftDescriptions: `{}`},
	}
	releases := []ConfigRelease{
		{
			EnvironmentID: 1,
			Version:       2,
			Config:        `{"service":{"host":"base"}}`,
			Descriptions:  `{}`,
		},
		{
			EnvironmentID: 2,
			Version:       4,
			Config:        `{"service":{"host":"old","port":8080}}`,
			Descriptions:  `{}`,
		},
	}

	if err := applyConfigEnvironmentReleaseState(environments, releases); err != nil {
		t.Fatalf("applyConfigEnvironmentReleaseState() error = %v", err)
	}
	if environments[0].HasDraft || environments[0].Version != 2 {
		t.Fatalf("base state = hasDraft %t, version %d; want false, 2", environments[0].HasDraft, environments[0].Version)
	}
	if !environments[1].HasDraft || environments[1].Version != 4 {
		t.Fatalf("child state = hasDraft %t, version %d; want true, 4", environments[1].HasDraft, environments[1].Version)
	}
	if !environments[2].HasDraft || environments[2].Version != 0 {
		t.Fatalf("unpublished state = hasDraft %t, version %d; want true, 0", environments[2].HasDraft, environments[2].Version)
	}
}

func TestValidateEnvironmentParentRejectsCycle(t *testing.T) {
	t.Parallel()

	environments := []ConfigEnvironment{
		{ID: 1, ParentID: 0},
		{ID: 2, ParentID: 1},
		{ID: 3, ParentID: 2},
	}
	err := validateEnvironmentParent(environments, 1, 3)
	if !errors.Is(err, ErrConfigEnvironmentInvalid) {
		t.Fatalf("validateEnvironmentParent() error = %v, want ErrConfigEnvironmentInvalid", err)
	}
}

func TestUniqueSortedIDs(t *testing.T) {
	t.Parallel()

	got := uniqueSortedIDs([]uint{4, 0, 2, 4, 1})
	want := []uint{1, 2, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("uniqueSortedIDs() = %v, want %v", got, want)
	}
}

func TestConfigNamespaceInputNormalizationAndValidation(t *testing.T) {
	t.Parallel()

	input := normalizeConfigNamespaceInput(ConfigNamespaceInput{
		Name:        "  Order Service  ",
		Slug:        "  ORDER-SERVICE  ",
		Description: "  order configuration  ",
	})
	if input.Name != "Order Service" || input.Slug != "order-service" || input.Description != "order configuration" {
		t.Fatalf("normalizeConfigNamespaceInput() = %#v", input)
	}
	if err := validateConfigNamespaceInput(input, false); err != nil {
		t.Fatalf("validateConfigNamespaceInput() error = %v", err)
	}

	invalid := input
	invalid.Slug = "invalid namespace"
	if err := validateConfigNamespaceInput(invalid, false); !errors.Is(err, ErrConfigNamespaceInvalid) {
		t.Fatalf("validateConfigNamespaceInput() error = %v, want ErrConfigNamespaceInvalid", err)
	}
}
