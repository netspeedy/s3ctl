// Package cli implements the s3ctl command-line application.
package cli

import (
	"fmt"
	"strings"
	"time"
)

const binaryName = "s3ctl"

const (
	providerS3                      = "s3"
	providerOVH                     = "ovh"
	operationProvision              = "provision"
	operationDelete                 = "delete"
	operationRepair                 = "repair"
	defaultProvider                 = providerS3
	defaultRegion                   = "us-east-1"
	defaultIAMUserPrefix            = ""
	defaultIAMPath                  = ""
	defaultCredentialPolicyTemplate = "bucket-readwrite"
	defaultOVHUserRole              = "objectstore_operator"
	defaultOVHStoragePolicyRole     = "readWrite"
	defaultConfigFileName           = "config.json"
	defaultOutputFormat             = "text"
	defaultProvisionTimeout         = 10 * time.Minute
	s3DeleteObjectsBatchSize        = 1000
)

type settings struct {
	ConfigPath               string            `json:"-"`
	Provider                 string            `json:"provider"`
	Bucket                   string            `json:"bucket"`
	Buckets                  []string          `json:"buckets"`
	BatchFile                string            `json:"batch_file"`
	Endpoint                 string            `json:"endpoint"`
	Region                   string            `json:"region"`
	Profile                  string            `json:"profile"`
	AccessKey                string            `json:"access_key"`
	SecretKey                string            `json:"secret_key"`
	SessionToken             string            `json:"session_token"`
	Insecure                 bool              `json:"insecure"`
	EnableVersioning         bool              `json:"enable_versioning"`
	BucketPolicyFile         string            `json:"bucket_policy_file"`
	BucketPolicyTemplate     string            `json:"bucket_policy_template"`
	CreateScopedCredentials  bool              `json:"create_scoped_credentials"`
	IAMEndpoint              string            `json:"iam_endpoint"`
	IAMUserName              string            `json:"iam_user_name"`
	IAMUserPrefix            string            `json:"iam_user_prefix"`
	IAMPath                  string            `json:"iam_path"`
	CredentialPolicyTemplate string            `json:"credential_policy_template"`
	OVHAPIEndpoint           string            `json:"ovh_api_endpoint"`
	OVHAccessToken           string            `json:"ovh_access_token"`
	OVHApplicationKey        string            `json:"ovh_application_key"`
	OVHApplicationSecret     string            `json:"ovh_application_secret"`
	OVHConsumerKey           string            `json:"ovh_consumer_key"`
	OVHClientID              string            `json:"ovh_client_id"`
	OVHClientSecret          string            `json:"ovh_client_secret"`
	OVHS3Endpoint            string            `json:"ovh_s3_endpoint"`
	OVHServiceName           string            `json:"ovh_service_name"`
	OVHUserRole              string            `json:"ovh_user_role"`
	OVHStoragePolicyRole     string            `json:"ovh_storage_policy_role"`
	OVHEncryptData           bool              `json:"ovh_encrypt_data"`
	OVHEncryptDataSet        bool              `json:"-"`
	OVHRotateCredentials     bool              `json:"ovh_rotate_credentials"`
	OVHRepairPolicies        bool              `json:"ovh_repair_policies"`
	OVHTags                  map[string]string `json:"ovh_tags"`
	DeleteBucket             bool              `json:"delete_bucket"`
	Force                    bool              `json:"force"`
	Timeout                  string            `json:"timeout"`
	Output                   string            `json:"output"`
	DryRun                   bool              `json:"dry_run"`
	ParsedTimeout            time.Duration
}

type source struct {
	Provider                 *string
	Buckets                  *[]string
	BatchFile                *string
	Endpoint                 *string
	Region                   *string
	Profile                  *string
	AccessKey                *string
	SecretKey                *string
	SessionToken             *string
	Insecure                 *bool
	EnableVersioning         *bool
	BucketPolicyFile         *string
	BucketPolicyTemplate     *string
	CreateScopedCredentials  *bool
	IAMEndpoint              *string
	IAMUserName              *string
	IAMUserPrefix            *string
	IAMPath                  *string
	CredentialPolicyTemplate *string
	OVHAPIEndpoint           *string
	OVHAccessToken           *string
	OVHApplicationKey        *string
	OVHApplicationSecret     *string
	OVHConsumerKey           *string
	OVHClientID              *string
	OVHClientSecret          *string
	OVHS3Endpoint            *string
	OVHServiceName           *string
	OVHUserRole              *string
	OVHStoragePolicyRole     *string
	OVHEncryptData           *bool
	OVHRotateCredentials     *bool
	OVHRepairPolicies        *bool
	OVHTags                  *map[string]string
	DeleteBucket             *bool
	Force                    *bool
	Timeout                  *time.Duration
	Output                   *string
	DryRun                   *bool
}

type cliFlags struct {
	Config                   string
	Provider                 string
	Buckets                  []string
	BatchFile                string
	Endpoint                 string
	Region                   string
	Profile                  string
	AccessKey                string
	SecretKey                string
	SessionToken             string
	Insecure                 bool
	EnableVersioning         bool
	BucketPolicyFile         string
	BucketPolicyTemplate     string
	CreateScopedCredentials  bool
	IAMEndpoint              string
	IAMUserName              string
	IAMUserPrefix            string
	IAMPath                  string
	CredentialPolicyTemplate string
	OVHAPIEndpoint           string
	OVHAccessToken           string
	OVHApplicationKey        string
	OVHApplicationSecret     string
	OVHConsumerKey           string
	OVHClientID              string
	OVHClientSecret          string
	OVHS3Endpoint            string
	OVHServiceName           string
	OVHUserRole              string
	OVHStoragePolicyRole     string
	OVHEncryptData           bool
	OVHRotateCredentials     bool
	OVHRepairPolicies        bool
	OVHTags                  []string
	DeleteBucket             bool
	Force                    bool
	Timeout                  string
	Output                   string
	DryRun                   bool
	Help                     bool
	HelpFull                 bool
	Version                  bool
}

type parseResult struct {
	source       source
	showHelp     bool
	showHelpFull bool
	showVersion  bool
}

type provisionTarget struct {
	Bucket                   string
	EnableVersioning         bool
	BucketPolicyFile         string
	BucketPolicyTemplate     string
	CreateScopedCredentials  bool
	IAMUserName              string
	CredentialPolicyTemplate string
}

type provisionResult struct {
	Operation     string           `json:"operation"`
	DryRun        bool             `json:"dry_run"`
	ConfigFile    string           `json:"config_file,omitempty"`
	ResourceCount int              `json:"resource_count"`
	Resources     []resourceResult `json:"resources"`
}

type commandErrorResult struct {
	Operation     string             `json:"operation"`
	DryRun        bool               `json:"dry_run"`
	ConfigFile    string             `json:"config_file,omitempty"`
	ResourceCount int                `json:"resource_count"`
	Error         commandErrorDetail `json:"error"`
}

type commandErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type resourceResult struct {
	BucketName          string                  `json:"bucket_name"`
	Endpoint            string                  `json:"endpoint,omitempty"`
	Region              string                  `json:"region"`
	Created             bool                    `json:"created"`
	Deleted             bool                    `json:"deleted,omitempty"`
	ObjectsDeleted      int                     `json:"objects_deleted,omitempty"`
	VersioningEnabled   bool                    `json:"versioning_enabled"`
	EncryptionEnabled   bool                    `json:"encryption_enabled"`
	BucketPolicyApplied bool                    `json:"bucket_policy_applied,omitempty"`
	BucketPolicySource  string                  `json:"bucket_policy_source,omitempty"`
	CredentialsRotated  bool                    `json:"credentials_rotated,omitempty"`
	CredentialsDeleted  int                     `json:"credentials_deleted,omitempty"`
	AccessPolicyApplied bool                    `json:"scoped_access_policy_applied,omitempty"`
	ScopedCredentials   *scopedCredentialResult `json:"scoped_credentials,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
}

type bucketExistsError struct {
	Name string
}

func (e bucketExistsError) Error() string {
	return fmt.Sprintf("bucket %q already exists", e.Name)
}

type bucketNotFoundError struct {
	Name     string
	Provider string
	Region   string
	Cause    error
}

func (e bucketNotFoundError) Error() string {
	if e.Provider == providerOVH {
		if strings.TrimSpace(e.Region) != "" {
			return fmt.Sprintf("OVH bucket/container %q does not exist in region %q; nothing was deleted", e.Name, e.Region)
		}
		return fmt.Sprintf("OVH bucket/container %q does not exist; nothing was deleted", e.Name)
	}
	return fmt.Sprintf("bucket %q does not exist; nothing was deleted", e.Name)
}

func (e bucketNotFoundError) Unwrap() error {
	return e.Cause
}
