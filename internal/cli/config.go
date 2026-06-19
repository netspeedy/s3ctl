package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

func resolveSettings(args []string, env map[string]string) (settings, parseResult, error) {
	cliParsed, err := parseFlags(args)
	if err != nil {
		return settings{}, parseResult{}, err
	}

	if cliParsed.showHelp || cliParsed.showHelpFull {
		return settings{}, cliParsed, nil
	}

	if cliParsed.showVersion {
		return mergeSources(source{}, cliParsed.source), cliParsed, nil
	}

	configPath, err := resolveConfigPath(args, env)
	if err != nil {
		return settings{}, parseResult{}, err
	}

	configSource, err := loadConfig(configPath)
	if err != nil {
		return settings{}, parseResult{}, err
	}

	cfg := mergeSources(configSource, cliParsed.source)
	cfg.ConfigPath = configPath
	if err := validateSettings(cfg); err != nil {
		return cfg, cliParsed, err
	}

	return cfg, cliParsed, nil
}

func parseFlags(args []string) (parseResult, error) {
	flags := cliFlags{}
	fs := newFlagSet(&flags)

	if err := fs.Parse(args); err != nil {
		return parseResult{}, err
	}
	timeout, err := changedDuration(fs, "timeout", flags.Timeout)
	if err != nil {
		return parseResult{}, err
	}
	ovhTags, err := changedStringMap(fs, "ovh-tag", flags.OVHTags)
	if err != nil {
		return parseResult{}, err
	}

	return parseResult{
		source: source{
			Provider:                 changedString(fs, "provider", flags.Provider),
			Buckets:                  changedStringSlice(fs, "bucket", flags.Buckets),
			BatchFile:                changedString(fs, "batch-file", flags.BatchFile),
			Endpoint:                 changedString(fs, "endpoint", flags.Endpoint),
			Region:                   changedString(fs, "region", flags.Region),
			Profile:                  changedString(fs, "profile", flags.Profile),
			AccessKey:                changedString(fs, "access-key", flags.AccessKey),
			SecretKey:                changedString(fs, "secret-key", flags.SecretKey),
			SessionToken:             changedString(fs, "session-token", flags.SessionToken),
			Insecure:                 changedBool(fs, "insecure", flags.Insecure),
			EnableVersioning:         changedBool(fs, "enable-versioning", flags.EnableVersioning),
			BucketPolicyFile:         changedString(fs, "bucket-policy-file", flags.BucketPolicyFile),
			BucketPolicyTemplate:     changedString(fs, "bucket-policy-template", flags.BucketPolicyTemplate),
			CreateScopedCredentials:  changedBool(fs, "create-scoped-credentials", flags.CreateScopedCredentials),
			IAMEndpoint:              changedString(fs, "iam-endpoint", flags.IAMEndpoint),
			IAMUserName:              changedString(fs, "iam-user-name", flags.IAMUserName),
			IAMUserPrefix:            changedString(fs, "iam-user-prefix", flags.IAMUserPrefix),
			IAMPath:                  changedString(fs, "iam-path", flags.IAMPath),
			CredentialPolicyTemplate: changedString(fs, "credential-policy-template", flags.CredentialPolicyTemplate),
			OVHAPIEndpoint:           changedString(fs, "ovh-api-endpoint", flags.OVHAPIEndpoint),
			OVHAccessToken:           changedString(fs, "ovh-access-token", flags.OVHAccessToken),
			OVHApplicationKey:        changedString(fs, "ovh-application-key", flags.OVHApplicationKey),
			OVHApplicationSecret:     changedString(fs, "ovh-application-secret", flags.OVHApplicationSecret),
			OVHConsumerKey:           changedString(fs, "ovh-consumer-key", flags.OVHConsumerKey),
			OVHClientID:              changedString(fs, "ovh-client-id", flags.OVHClientID),
			OVHClientSecret:          changedString(fs, "ovh-client-secret", flags.OVHClientSecret),
			OVHS3Endpoint:            changedString(fs, "ovh-s3-endpoint", flags.OVHS3Endpoint),
			OVHServiceName:           changedString(fs, "ovh-service-name", flags.OVHServiceName),
			OVHUserRole:              changedString(fs, "ovh-user-role", flags.OVHUserRole),
			OVHStoragePolicyRole:     changedString(fs, "ovh-storage-policy-role", flags.OVHStoragePolicyRole),
			OVHEncryptData:           changedBool(fs, "ovh-encrypt-data", flags.OVHEncryptData),
			OVHRotateCredentials:     changedBool(fs, "ovh-rotate-credentials", flags.OVHRotateCredentials),
			OVHRepairPolicies:        changedBool(fs, "ovh-repair-policies", flags.OVHRepairPolicies),
			OVHTags:                  ovhTags,
			DeleteBucket:             changedBool(fs, "delete", flags.DeleteBucket),
			Force:                    changedBool(fs, "force", flags.Force),
			Timeout:                  timeout,
			Output:                   changedString(fs, "output", flags.Output),
			DryRun:                   changedBool(fs, "dry-run", flags.DryRun),
		},
		showHelp:     flags.Help,
		showHelpFull: flags.HelpFull,
		showVersion:  flags.Version,
	}, nil
}

func loadConfig(path string) (source, error) {
	if path == "" {
		return source{}, nil
	}

	if filepath.Ext(path) != ".json" {
		return source{}, fmt.Errorf("config file must end with .json: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return source{}, err
	}

	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return source{}, err
	}

	configDir := filepath.Dir(path)
	batchFile := resolveRelativePathIfSet(configDir, cfg.BatchFile)
	bucketPolicyFile := resolveRelativePathIfSet(configDir, cfg.BucketPolicyFile)

	buckets := make([]string, 0, len(cfg.Buckets)+1)
	if strings.TrimSpace(cfg.Bucket) != "" {
		buckets = append(buckets, cfg.Bucket)
	}
	buckets = append(buckets, cfg.Buckets...)
	deleteBucket := boolPtrIfSet(data, "delete_bucket", cfg.DeleteBucket)
	if deleteBucket == nil {
		var err error
		deleteBucket, err = boolPtrFromJSONField(data, "delete")
		if err != nil {
			return source{}, err
		}
	}
	timeout, err := durationPtrFromJSONFields(data, "timeout", "provision_timeout")
	if err != nil {
		return source{}, err
	}

	return source{
		Provider:                 stringPtrIfField(data, "provider", cfg.Provider),
		Buckets:                  stringSlicePtrIfSet(data, []string{"bucket", "buckets"}, buckets),
		BatchFile:                stringPtrIfField(data, "batch_file", batchFile),
		Endpoint:                 stringPtrIfField(data, "endpoint", cfg.Endpoint),
		Region:                   stringPtrIfField(data, "region", cfg.Region),
		Profile:                  stringPtrIfField(data, "profile", cfg.Profile),
		AccessKey:                stringPtrIfField(data, "access_key", cfg.AccessKey),
		SecretKey:                stringPtrIfField(data, "secret_key", cfg.SecretKey),
		SessionToken:             stringPtrIfField(data, "session_token", cfg.SessionToken),
		Insecure:                 boolPtrIfSet(data, "insecure", cfg.Insecure),
		EnableVersioning:         boolPtrIfSet(data, "enable_versioning", cfg.EnableVersioning),
		BucketPolicyFile:         stringPtrIfField(data, "bucket_policy_file", bucketPolicyFile),
		BucketPolicyTemplate:     stringPtrIfField(data, "bucket_policy_template", cfg.BucketPolicyTemplate),
		CreateScopedCredentials:  boolPtrIfSet(data, "create_scoped_credentials", cfg.CreateScopedCredentials),
		IAMEndpoint:              stringPtrIfField(data, "iam_endpoint", cfg.IAMEndpoint),
		IAMUserName:              stringPtrIfField(data, "iam_user_name", cfg.IAMUserName),
		IAMUserPrefix:            stringPtrIfField(data, "iam_user_prefix", cfg.IAMUserPrefix),
		IAMPath:                  stringPtrIfField(data, "iam_path", cfg.IAMPath),
		CredentialPolicyTemplate: stringPtrIfField(data, "credential_policy_template", cfg.CredentialPolicyTemplate),
		OVHAPIEndpoint:           stringPtrIfField(data, "ovh_api_endpoint", cfg.OVHAPIEndpoint),
		OVHAccessToken:           stringPtrIfField(data, "ovh_access_token", cfg.OVHAccessToken),
		OVHApplicationKey:        stringPtrIfField(data, "ovh_application_key", cfg.OVHApplicationKey),
		OVHApplicationSecret:     stringPtrIfField(data, "ovh_application_secret", cfg.OVHApplicationSecret),
		OVHConsumerKey:           stringPtrIfField(data, "ovh_consumer_key", cfg.OVHConsumerKey),
		OVHClientID:              stringPtrIfField(data, "ovh_client_id", cfg.OVHClientID),
		OVHClientSecret:          stringPtrIfField(data, "ovh_client_secret", cfg.OVHClientSecret),
		OVHS3Endpoint:            stringPtrIfField(data, "ovh_s3_endpoint", cfg.OVHS3Endpoint),
		OVHServiceName:           stringPtrIfField(data, "ovh_service_name", cfg.OVHServiceName),
		OVHUserRole:              stringPtrIfField(data, "ovh_user_role", cfg.OVHUserRole),
		OVHStoragePolicyRole:     stringPtrIfField(data, "ovh_storage_policy_role", cfg.OVHStoragePolicyRole),
		OVHEncryptData:           boolPtrIfSet(data, "ovh_encrypt_data", cfg.OVHEncryptData),
		OVHRotateCredentials:     boolPtrIfSet(data, "ovh_rotate_credentials", cfg.OVHRotateCredentials),
		OVHRepairPolicies:        boolPtrIfSet(data, "ovh_repair_policies", cfg.OVHRepairPolicies),
		OVHTags:                  stringMapPtrIfField(data, "ovh_tags", cfg.OVHTags),
		DeleteBucket:             deleteBucket,
		Force:                    boolPtrIfSet(data, "force", cfg.Force),
		Timeout:                  timeout,
		Output:                   stringPtrIfField(data, "output", cfg.Output),
		DryRun:                   boolPtrIfSet(data, "dry_run", cfg.DryRun),
	}, nil
}

func mergeSources(sources ...source) settings {
	cfg := settings{
		Provider:                 defaultProvider,
		Region:                   defaultRegion,
		IAMUserPrefix:            defaultIAMUserPrefix,
		IAMPath:                  defaultIAMPath,
		CredentialPolicyTemplate: defaultCredentialPolicyTemplate,
		OVHUserRole:              defaultOVHUserRole,
		OVHStoragePolicyRole:     defaultOVHStoragePolicyRole,
		Output:                   defaultOutputFormat,
		ParsedTimeout:            defaultProvisionTimeout,
	}

	for _, src := range sources {
		if src.Provider != nil {
			cfg.Provider = *src.Provider
		}
		if src.Profile != nil {
			cfg.Profile = *src.Profile
			cfg.AccessKey = ""
			cfg.SecretKey = ""
			cfg.SessionToken = ""
		}
		if src.AccessKey != nil || src.SecretKey != nil || src.SessionToken != nil {
			cfg.Profile = ""
		}
		if src.Buckets != nil {
			cfg.Buckets = append([]string{}, (*src.Buckets)...)
		}
		if src.BatchFile != nil {
			cfg.BatchFile = *src.BatchFile
		}
		if src.Endpoint != nil {
			cfg.Endpoint = *src.Endpoint
		}
		if src.Region != nil {
			cfg.Region = *src.Region
		}
		if src.AccessKey != nil {
			cfg.AccessKey = *src.AccessKey
		}
		if src.SecretKey != nil {
			cfg.SecretKey = *src.SecretKey
		}
		if src.SessionToken != nil {
			cfg.SessionToken = *src.SessionToken
		}
		if src.Insecure != nil {
			cfg.Insecure = *src.Insecure
		}
		if src.EnableVersioning != nil {
			cfg.EnableVersioning = *src.EnableVersioning
		}
		if src.BucketPolicyFile != nil {
			cfg.BucketPolicyFile = *src.BucketPolicyFile
		}
		if src.BucketPolicyTemplate != nil {
			cfg.BucketPolicyTemplate = *src.BucketPolicyTemplate
		}
		if src.CreateScopedCredentials != nil {
			cfg.CreateScopedCredentials = *src.CreateScopedCredentials
		}
		if src.IAMEndpoint != nil {
			cfg.IAMEndpoint = *src.IAMEndpoint
		}
		if src.IAMUserName != nil {
			cfg.IAMUserName = *src.IAMUserName
		}
		if src.IAMUserPrefix != nil {
			cfg.IAMUserPrefix = *src.IAMUserPrefix
		}
		if src.IAMPath != nil {
			cfg.IAMPath = *src.IAMPath
		}
		if src.CredentialPolicyTemplate != nil {
			cfg.CredentialPolicyTemplate = *src.CredentialPolicyTemplate
		}
		if src.OVHAPIEndpoint != nil {
			cfg.OVHAPIEndpoint = *src.OVHAPIEndpoint
		}
		if src.OVHAccessToken != nil {
			cfg.OVHAccessToken = *src.OVHAccessToken
		}
		if src.OVHApplicationKey != nil {
			cfg.OVHApplicationKey = *src.OVHApplicationKey
		}
		if src.OVHApplicationSecret != nil {
			cfg.OVHApplicationSecret = *src.OVHApplicationSecret
		}
		if src.OVHConsumerKey != nil {
			cfg.OVHConsumerKey = *src.OVHConsumerKey
		}
		if src.OVHClientID != nil {
			cfg.OVHClientID = *src.OVHClientID
		}
		if src.OVHClientSecret != nil {
			cfg.OVHClientSecret = *src.OVHClientSecret
		}
		if src.OVHS3Endpoint != nil {
			cfg.OVHS3Endpoint = *src.OVHS3Endpoint
		}
		if src.OVHServiceName != nil {
			cfg.OVHServiceName = *src.OVHServiceName
		}
		if src.OVHUserRole != nil {
			cfg.OVHUserRole = *src.OVHUserRole
		}
		if src.OVHStoragePolicyRole != nil {
			cfg.OVHStoragePolicyRole = *src.OVHStoragePolicyRole
		}
		if src.OVHEncryptData != nil {
			cfg.OVHEncryptData = *src.OVHEncryptData
			cfg.OVHEncryptDataSet = true
		}
		if src.OVHRotateCredentials != nil {
			cfg.OVHRotateCredentials = *src.OVHRotateCredentials
		}
		if src.OVHRepairPolicies != nil {
			cfg.OVHRepairPolicies = *src.OVHRepairPolicies
		}
		if src.OVHTags != nil {
			cfg.OVHTags = cloneStringMap(*src.OVHTags)
		}
		if src.DeleteBucket != nil {
			cfg.DeleteBucket = *src.DeleteBucket
		}
		if src.Force != nil {
			cfg.Force = *src.Force
		}
		if src.Timeout != nil {
			cfg.ParsedTimeout = *src.Timeout
			cfg.Timeout = src.Timeout.String()
		}
		if src.Output != nil {
			cfg.Output = *src.Output
		}
		if src.DryRun != nil {
			cfg.DryRun = *src.DryRun
		}
	}

	cfg.Buckets = dedupeStringsPreserveOrder(cfg.Buckets)
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	cfg.OVHStoragePolicyRole = normalizeOVHStoragePolicyRole(cfg.OVHStoragePolicyRole)
	return cfg
}

func validateSettings(cfg settings) error {
	if len(cfg.Buckets) == 0 && cfg.BatchFile == "" {
		return errors.New("at least one --bucket or a --batch-file is required unless provided via config")
	}
	provider := cfg.Provider
	if provider == "" {
		provider = defaultProvider
	}
	switch provider {
	case providerS3, providerOVH:
	default:
		return fmt.Errorf("--provider must be either s3 or ovh, got %q", cfg.Provider)
	}
	if cfg.OVHRotateCredentials && provider != providerOVH {
		return errors.New("--ovh-rotate-credentials requires --provider ovh")
	}
	if cfg.OVHRepairPolicies && provider != providerOVH {
		return errors.New("--ovh-repair-policies requires --provider ovh")
	}
	if len(cfg.OVHTags) > 0 && provider != providerOVH {
		return errors.New("--ovh-tag and ovh_tags require --provider ovh")
	}
	if err := validateStringMap("OVH tag", cfg.OVHTags); err != nil {
		return err
	}
	if cfg.DeleteBucket && cfg.OVHRotateCredentials {
		return errors.New("use either --delete or --ovh-rotate-credentials, not both")
	}
	if cfg.DeleteBucket && cfg.OVHRepairPolicies {
		return errors.New("use either --delete or --ovh-repair-policies, not both")
	}
	if cfg.OVHRotateCredentials && cfg.OVHRepairPolicies {
		return errors.New("use either --ovh-rotate-credentials or --ovh-repair-policies, not both")
	}
	if cfg.BucketPolicyFile != "" && cfg.BucketPolicyTemplate != "" {
		return errors.New("use either --bucket-policy-file or --bucket-policy-template, not both")
	}
	if cfg.AccessKey != "" && cfg.SecretKey == "" {
		return errors.New("--access-key and --secret-key must be provided together")
	}
	if cfg.AccessKey == "" && cfg.SecretKey != "" {
		return errors.New("--access-key and --secret-key must be provided together")
	}
	if cfg.SessionToken != "" && (cfg.AccessKey == "" || cfg.SecretKey == "") {
		return errors.New("--session-token requires --access-key and --secret-key")
	}
	if cfg.Profile != "" && (cfg.AccessKey != "" || cfg.SecretKey != "" || cfg.SessionToken != "") {
		return errors.New("use either --profile or explicit credentials, not both")
	}
	output := cfg.Output
	if output == "" {
		output = defaultOutputFormat
	}
	if output != "text" && output != "json" {
		return fmt.Errorf("--output must be either text or json, got %q", cfg.Output)
	}
	if cfg.BucketPolicyTemplate != "" {
		if _, ok := bucketPolicyTemplates[cfg.BucketPolicyTemplate]; !ok {
			return fmt.Errorf("unsupported bucket policy template %q", cfg.BucketPolicyTemplate)
		}
	}
	if cfg.CredentialPolicyTemplate != "" {
		if _, ok := credentialPolicyTemplates[cfg.CredentialPolicyTemplate]; !ok {
			return fmt.Errorf("unsupported credential policy template %q", cfg.CredentialPolicyTemplate)
		}
	}
	if cfg.IAMUserName != "" && !cfg.CreateScopedCredentials {
		return errors.New("--iam-user-name requires --create-scoped-credentials")
	}
	if provider == providerOVH {
		if err := validateOVHSettings(cfg); err != nil {
			return err
		}
	}
	return nil
}

func stringPtrIfField(data []byte, field, value string) *string {
	if !jsonFieldPresent(data, field) {
		return nil
	}
	valueCopy := value
	return &valueCopy
}

func stringSlicePtrIfSet(data []byte, fields, values []string) *[]string {
	for _, field := range fields {
		if jsonFieldPresent(data, field) {
			valueCopy := append([]string{}, dedupeStringsPreserveOrder(values)...)
			return &valueCopy
		}
	}
	return nil
}

func stringMapPtrIfField(data []byte, field string, value map[string]string) *map[string]string {
	if !jsonFieldPresent(data, field) {
		return nil
	}
	valueCopy := cloneStringMap(value)
	return &valueCopy
}

func boolPtrIfSet(data []byte, field string, value bool) *bool {
	if !jsonFieldPresent(data, field) {
		return nil
	}
	valueCopy := value
	return &valueCopy
}

func boolPtrFromJSONField(data []byte, field string) (*bool, error) {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	raw, ok := decoded[field]
	if !ok {
		return nil, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("config field %s must be a boolean", field)
	}
	return &value, nil
}

func durationPtrFromJSONFields(data []byte, fields ...string) (*time.Duration, error) {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	for _, field := range fields {
		raw, ok := decoded[field]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("config field %s must be a duration string", field)
		}
		parsed, err := parsePositiveDuration(value)
		if err != nil {
			return nil, fmt.Errorf("config field %s must be a positive duration", field)
		}
		return &parsed, nil
	}
	return nil, nil
}

func jsonFieldPresent(data []byte, field string) bool {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		return false
	}
	_, ok := decoded[field]
	return ok
}

func changedString(fs *pflag.FlagSet, name, value string) *string {
	if fs.Changed(name) {
		valueCopy := value
		return &valueCopy
	}
	return nil
}

func changedStringSlice(fs *pflag.FlagSet, name string, values []string) *[]string {
	if fs.Changed(name) {
		valueCopy := append([]string{}, values...)
		return &valueCopy
	}
	return nil
}

func changedStringMap(fs *pflag.FlagSet, name string, values []string) (*map[string]string, error) {
	if !fs.Changed(name) {
		return nil, nil
	}
	parsed, err := parseStringMap(values)
	if err != nil {
		return nil, fmt.Errorf("--%s must be key=value: %w", name, err)
	}
	return &parsed, nil
}

func changedBool(fs *pflag.FlagSet, name string, value bool) *bool {
	if fs.Changed(name) {
		valueCopy := value
		return &valueCopy
	}
	return nil
}

func changedDuration(fs *pflag.FlagSet, name, value string) (*time.Duration, error) {
	if !fs.Changed(name) {
		return nil, nil
	}
	parsed, err := parsePositiveDuration(value)
	if err != nil {
		return nil, fmt.Errorf("--%s must be a positive duration such as 30s, 5m, or 1h", name)
	}
	return &parsed, nil
}

func parsePositiveDuration(value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, errors.New("duration must be positive")
	}
	return parsed, nil
}

func parseStringMap(values []string) (map[string]string, error) {
	parsed := make(map[string]string, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key, val, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("%q is missing =", value)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			return nil, fmt.Errorf("%q has an empty key", value)
		}
		if val == "" {
			return nil, fmt.Errorf("%q has an empty value", value)
		}
		parsed[key] = val
	}
	return parsed, nil
}

func validateStringMap(label string, values map[string]string) error {
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s key must not be empty", label)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s %q value must not be empty", label, key)
		}
	}
	return nil
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func extractConfigPath(args []string) string {
	for index := 0; index < len(args); index++ {
		if args[index] == "--config" && index+1 < len(args) {
			return args[index+1]
		}
		if args[index] == "-c" && index+1 < len(args) {
			return args[index+1]
		}
		if strings.HasPrefix(args[index], "--config=") {
			return strings.TrimPrefix(args[index], "--config=")
		}
		if strings.HasPrefix(args[index], "-c=") {
			return strings.TrimPrefix(args[index], "-c=")
		}
	}
	return ""
}

func extractOutputFormat(args []string) string {
	output := ""
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--output" && index+1 < len(args):
			output = args[index+1]
		case arg == "-o" && index+1 < len(args):
			output = args[index+1]
		case strings.HasPrefix(arg, "--output="):
			output = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-o="):
			output = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "-o") && !strings.HasPrefix(arg, "--") && len(arg) > len("-o"):
			output = strings.TrimPrefix(arg, "-o")
		}
	}
	return output
}

func detectOutputFormat(args []string, env map[string]string) string {
	output := defaultOutputFormat
	if configPath, err := resolveConfigPath(args, env); err == nil && configPath != "" {
		if configOutput := readConfigOutputFormat(configPath); configOutput != "" {
			output = configOutput
		}
	}
	if cliOutput := extractOutputFormat(args); cliOutput != "" {
		output = cliOutput
	}
	return strings.ToLower(strings.TrimSpace(output))
}

func readConfigOutputFormat(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		return ""
	}

	raw, ok := decoded["output"]
	if !ok {
		return ""
	}
	var output string
	if err := json.Unmarshal(raw, &output); err != nil {
		return ""
	}
	return output
}

func resolveConfigPath(args []string, env map[string]string) (string, error) {
	if configPath := extractConfigPath(args); configPath != "" {
		return configPath, nil
	}

	configPath := defaultConfigPath(env)
	if configPath == "" {
		return "", nil
	}

	info, err := os.Stat(configPath)
	if err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("default config path %s is a directory", configPath)
		}
		return configPath, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	return "", err
}

func defaultConfigPath(env map[string]string) string {
	baseDir := strings.TrimSpace(env["XDG_CONFIG_HOME"])
	if baseDir == "" {
		homeDir := strings.TrimSpace(env["HOME"])
		if homeDir == "" {
			resolvedHomeDir, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			homeDir = resolvedHomeDir
		}
		baseDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(baseDir, binaryName, defaultConfigFileName)
}

func dedupeStringsPreserveOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
