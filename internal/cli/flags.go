package cli

import (
	"io"

	"github.com/spf13/pflag"
)

func newFlagSet(flags *cliFlags) *pflag.FlagSet {
	fs := pflag.NewFlagSet(binaryName, pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.SortFlags = false

	fs.StringVarP(&flags.Config, "config", "c", "", "Path to a JSON config file")
	fs.StringVar(&flags.Provider, "provider", defaultProvider, "Provisioning provider: s3 or ovh")
	fs.StringArrayVarP(&flags.Buckets, "bucket", "b", nil, "Bucket name to create or delete; may be specified more than once")
	fs.StringVar(&flags.BatchFile, "batch-file", "", "Path to a CSV file describing multiple bucket requests")
	fs.StringVar(&flags.Endpoint, "endpoint", "", "S3 endpoint URL for S3-compatible services")
	fs.StringVar(&flags.Region, "region", defaultRegion, "Bucket region")
	fs.StringVar(&flags.Profile, "profile", "", "AWS profile name for SDK configuration")
	fs.StringVar(&flags.AccessKey, "access-key", "", "Access key for the S3 API client")
	fs.StringVar(&flags.SecretKey, "secret-key", "", "Secret key for the S3 API client")
	fs.StringVar(&flags.SessionToken, "session-token", "", "Optional session token for the S3 API client")
	fs.BoolVar(&flags.Insecure, "insecure", false, "Disable TLS certificate verification")
	fs.BoolVar(&flags.EnableVersioning, "enable-versioning", false, "Enable bucket versioning after creation")
	fs.StringVar(&flags.BucketPolicyFile, "bucket-policy-file", "", "Path to a JSON bucket policy document")
	fs.StringVar(&flags.BucketPolicyTemplate, "bucket-policy-template", "", "Built-in bucket policy template")
	fs.BoolVar(&flags.CreateScopedCredentials, "create-scoped-credentials", false, "Create a new scoped IAM-style user and access key for each bucket")
	fs.StringVar(&flags.IAMEndpoint, "iam-endpoint", "", "Override the IAM API endpoint used for scoped credential provisioning")
	fs.StringVar(&flags.IAMUserName, "iam-user-name", "", "Explicit IAM user name for a single bucket run")
	fs.StringVar(&flags.IAMUserPrefix, "iam-user-prefix", defaultIAMUserPrefix, "Optional prefix used when generating IAM user names automatically")
	fs.StringVar(&flags.IAMPath, "iam-path", defaultIAMPath, "Optional IAM path used for generated users")
	fs.StringVar(&flags.CredentialPolicyTemplate, "credential-policy-template", defaultCredentialPolicyTemplate, "Built-in scoped credential policy template")
	fs.StringVar(&flags.OVHAPIEndpoint, "ovh-api-endpoint", "", "OVHcloud API endpoint name or URL for the OVH provider")
	fs.StringVar(&flags.OVHAccessToken, "ovh-access-token", "", "OVHcloud access token for the OVH provider")
	fs.StringVar(&flags.OVHApplicationKey, "ovh-application-key", "", "OVHcloud application key for the OVH provider")
	fs.StringVar(&flags.OVHApplicationSecret, "ovh-application-secret", "", "OVHcloud application secret for the OVH provider")
	fs.StringVar(&flags.OVHConsumerKey, "ovh-consumer-key", "", "OVHcloud consumer key for the OVH provider")
	fs.StringVar(&flags.OVHClientID, "ovh-client-id", "", "OVHcloud OAuth2 client ID for the OVH provider")
	fs.StringVar(&flags.OVHClientSecret, "ovh-client-secret", "", "OVHcloud OAuth2 client secret for the OVH provider")
	fs.StringVar(&flags.OVHS3Endpoint, "ovh-s3-endpoint", "", "Override the returned OVHcloud S3 endpoint URL")
	fs.StringVar(&flags.OVHServiceName, "ovh-service-name", "", "OVHcloud Public Cloud project service name for the OVH provider")
	fs.StringVar(&flags.OVHUserRole, "ovh-user-role", defaultOVHUserRole, "OVHcloud Public Cloud user role for created object storage users")
	fs.StringVar(&flags.OVHStoragePolicyRole, "ovh-storage-policy-role", defaultOVHStoragePolicyRole, "OVHcloud access policy role: admin, deny, readOnly, readWrite, or replication")
	fs.BoolVar(&flags.OVHEncryptData, "ovh-encrypt-data", false, "Enable OVHcloud server-side encryption with OVH-managed keys")
	fs.BoolVar(&flags.OVHRotateCredentials, "ovh-rotate-credentials", false, "Rotate existing OVHcloud S3 credentials for each bucket instead of creating containers")
	fs.BoolVar(&flags.OVHRepairPolicies, "ovh-repair-policies", false, "Apply scoped OVHcloud S3 and container policies to existing bucket users without rotating credentials")
	fs.StringArrayVar(&flags.OVHTags, "ovh-tag", nil, "Tag to apply to OVHcloud containers as key=value; may be specified more than once")
	fs.BoolVar(&flags.DeleteBucket, "delete", false, "Delete each bucket instead of creating buckets")
	fs.BoolVar(&flags.Force, "force", false, "Allow delete operations to remove bucket contents before deleting buckets")
	fs.StringVar(&flags.Timeout, "timeout", defaultProvisionTimeout.String(), "Overall operation timeout, for example 30s, 5m, or 1h")
	fs.StringVarP(&flags.Output, "output", "o", defaultOutputFormat, "Output format: text or json")
	fs.BoolVar(&flags.DryRun, "dry-run", false, "Show the planned actions without making changes")
	fs.BoolVarP(&flags.Help, "help", "h", false, "Show help")
	fs.BoolVar(&flags.HelpFull, "help-full", false, "Show the full reference help")
	fs.BoolVar(&flags.Version, "version", false, "Show version information")

	return fs
}

type helpFlagValue struct {
	pflag.Value
	valueType string
}

func (value helpFlagValue) Type() string {
	return value.valueType
}

func setHelpValueTypes(fs *pflag.FlagSet) {
	valueTypes := map[string]string{
		"access-key":                 "ID",
		"batch-file":                 "PATH",
		"bucket":                     "NAME",
		"bucket-policy-file":         "PATH",
		"bucket-policy-template":     "NAME",
		"config":                     "PATH",
		"credential-policy-template": "NAME",
		"endpoint":                   "URL",
		"iam-endpoint":               "URL",
		"iam-path":                   "PATH",
		"iam-user-name":              "NAME",
		"iam-user-prefix":            "PREFIX",
		"output":                     "FORMAT",
		"ovh-access-token":           "TOKEN",
		"ovh-api-endpoint":           "URL",
		"ovh-application-key":        "KEY",
		"ovh-application-secret":     "SECRET",
		"ovh-client-id":              "ID",
		"ovh-client-secret":          "SECRET",
		"ovh-consumer-key":           "KEY",
		"ovh-s3-endpoint":            "URL",
		"ovh-service-name":           "PROJECT_ID",
		"ovh-storage-policy-role":    "ROLE",
		"ovh-tag":                    "KEY=VALUE",
		"ovh-user-role":              "ROLE",
		"profile":                    "NAME",
		"provider":                   "NAME",
		"region":                     "NAME",
		"secret-key":                 "SECRET",
		"session-token":              "TOKEN",
		"timeout":                    "DURATION",
	}
	for name, valueType := range valueTypes {
		flag := fs.Lookup(name)
		if flag == nil {
			continue
		}
		flag.Value = helpFlagValue{Value: flag.Value, valueType: valueType}
		if name == "bucket" || name == "ovh-tag" {
			flag.DefValue = ""
		}
	}
}
