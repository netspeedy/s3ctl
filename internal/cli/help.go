package cli

import (
	"fmt"
	"io"
	"strings"
)

func writeUsage(w io.Writer) error {
	_, err := io.WriteString(w, usageText())
	return err
}

func writeUsageForArgs(w io.Writer, args []string) error {
	_, err := io.WriteString(w, usageTextForArgs(args))
	return err
}

func usageTextForArgs(args []string) string {
	if argHasFlag(args, "help-full", "") {
		return fullUsageText()
	}
	if shouldShowBucketHelp(args) {
		return bucketUsageText()
	}
	return usageText()
}

func usageText() string {
	return fmt.Sprintf(`%s creates S3 buckets and bucket-scoped credentials.

Usage:
  %s --bucket NAME [options]
  %s --batch-file PATH [options]
  %s --config PATH [options]

Common workflows:
  %s --bucket app-data --dry-run
  %s --bucket app-data --create-scoped-credentials --output json
  %s --provider ovh --bucket app-data --region UK --ovh-service-name PROJECT_ID
  %s --provider ovh --bucket app-data --ovh-rotate-credentials --output json
  %s --bucket app-data --delete
  %s --bucket app-data --delete --force --timeout 30m

Core options:
  -b, --bucket NAME            Bucket to create, rotate, or delete. Repeatable.
      --batch-file PATH        CSV file of bucket requests.
  -c, --config PATH            JSON config file.
      --provider NAME          Provider: s3 or ovh. Default: s3.
      --endpoint URL           S3-compatible endpoint URL.
      --region NAME            Bucket region. Default: us-east-1.
  -o, --output FORMAT          Output: text or json. Default: text.
      --dry-run                Show planned actions without making changes.
      --timeout DURATION       Overall timeout. Default: 10m.

Bucket options:
      --enable-versioning
      --bucket-policy-file PATH
      --bucket-policy-template NAME
      --create-scoped-credentials
      --credential-policy-template NAME

S3/IAM options:
      --profile NAME
      --access-key ID
      --secret-key SECRET
      --session-token TOKEN
      --iam-endpoint URL
      --iam-user-prefix PREFIX

OVHcloud options:
      --ovh-service-name PROJECT_ID
      --ovh-client-id ID
      --ovh-client-secret SECRET
      --ovh-application-key KEY
      --ovh-application-secret SECRET
      --ovh-consumer-key KEY
      --ovh-encrypt-data
      --ovh-rotate-credentials
      --ovh-repair-policies
      --ovh-tag KEY=VALUE

Delete options:
      --delete                 Delete buckets instead of creating them.
      --force                  Empty non-empty buckets before delete.

More help:
  %s --bucket NAME --help      Show bucket workflow help.
  %s --help-full               Show every flag, template, and CSV field.
  %s --version                 Show version information.
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func fullUsageText() string {
	flags := cliFlags{}
	fs := newFlagSet(&flags)
	setHelpValueTypes(fs)
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, `%s full reference.

Usage:
  %s [options]

Examples:
  %s --bucket app-data --endpoint https://objects.example.com --region us-east-1
  %s --provider ovh --bucket app-data --region GRA --ovh-service-name PROJECT_ID
  %s --provider ovh --bucket app-data --ovh-rotate-credentials --output json
  %s --provider ovh --bucket app-data --ovh-repair-policies --output json
  %s --provider ovh --bucket app-data --delete
  %s --provider ovh --bucket app-data --delete --force
  %s --bucket app-data --create-scoped-credentials --credential-policy-template bucket-readwrite
  %s --bucket app-data --bucket logs --create-scoped-credentials --dry-run --output json
  %s --batch-file ./examples/aws/s3ctl-batch.csv --create-scoped-credentials
  %s --bucket app-data --dry-run
  %s --config ./examples/aws/s3ctl.json --dry-run --output json

Flags:
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
	builder.WriteString(fs.FlagUsagesWrapped(100))

	builder.WriteString(`
Configuration precedence:
  1. CLI flags
  2. JSON config file
  3. Built-in defaults

Default user config file:
  $XDG_CONFIG_HOME/s3ctl/config.json
  $HOME/.config/s3ctl/config.json

Built-in bucket policy templates:
`)
	for _, name := range sortedKeys(bucketPolicyTemplates) {
		_, _ = fmt.Fprintf(&builder, "  %s\n      %s\n", name, bucketPolicyTemplates[name])
	}

	builder.WriteString(`
Built-in scoped credential policy templates:
`)
	for _, name := range sortedKeys(credentialPolicyTemplates) {
		_, _ = fmt.Fprintf(&builder, "  %s\n      %s\n", name, credentialPolicyTemplates[name])
	}

	builder.WriteString(`
Batch CSV columns:
  bucket
  iam_user_name
  enable_versioning
  bucket_policy_file
  bucket_policy_template
  create_scoped_credentials
  credential_policy_template

Notes:
  The default provider is s3, which provisions through the S3 API.
  Scoped credential provisioning for the s3 provider uses the IAM API. By default this targets AWS IAM.
  Use --iam-endpoint when you need a different IAM-compatible endpoint.
  Use --provider ovh to create OVHcloud Public Cloud users, S3 credentials, and containers through the OVHcloud API.
  Standard AWS SDK credential and profile discovery is used when --profile or explicit access key values are not set.
  Standard go-ovh client discovery, including ovh.conf, is used when explicit OVH auth flags or config values are not set.
`)

	return builder.String()
}

func bucketUsageText() string {
	return fmt.Sprintf(`%s bucket workflow help.

Usage:
  %s --bucket NAME [options]
  %s --bucket NAME --delete [options]
  %s --batch-file PATH [options]

Create:
  %s --bucket app-data --dry-run
  %s --bucket app-data --create-scoped-credentials --output json
  %s --provider ovh --bucket app-data --region UK --ovh-service-name PROJECT_ID

Rotate:
  %s --provider ovh --bucket app-data --ovh-rotate-credentials --output json

Delete:
  %s --bucket app-data --delete
  %s --bucket app-data --delete --force --timeout 30m

Bucket workflow options:
  -b, --bucket NAME            Bucket target. Repeatable.
      --batch-file PATH        CSV file of bucket targets.
      --provider NAME          Provider: s3 or ovh. Default: s3.
      --region NAME            Bucket region.
      --endpoint URL           S3-compatible endpoint URL.
      --enable-versioning
      --create-scoped-credentials
      --credential-policy-template NAME
      --bucket-policy-file PATH
      --bucket-policy-template NAME

OVH bucket options:
      --ovh-service-name PROJECT_ID
      --ovh-storage-policy-role ROLE
      --ovh-encrypt-data
      --ovh-rotate-credentials
      --ovh-repair-policies
      --ovh-tag KEY=VALUE

Delete options:
      --delete                 Delete buckets instead of creating them.
      --force                  Empty non-empty buckets before delete.
      --dry-run                Show planned actions without making changes.
      --timeout DURATION       Overall timeout.

Output and config:
  -o, --output FORMAT          Output: text or json.
  -c, --config PATH            JSON config file.

More help:
  %s --help-full               Show every provider, auth, IAM, and configuration option.
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func shouldShowBucketHelp(args []string) bool {
	for _, flag := range []struct {
		name      string
		shorthand string
	}{
		{name: "bucket", shorthand: "b"},
		{name: "batch-file"},
		{name: "delete"},
		{name: "force"},
		{name: "enable-versioning"},
		{name: "bucket-policy-file"},
		{name: "bucket-policy-template"},
		{name: "create-scoped-credentials"},
		{name: "ovh-rotate-credentials"},
		{name: "ovh-repair-policies"},
		{name: "ovh-tag"},
	} {
		if argHasFlag(args, flag.name, flag.shorthand) {
			return true
		}
	}
	return false
}

func argHasFlag(args []string, name, shorthand string) bool {
	longFlag := "--" + name
	shortFlag := "-" + shorthand
	for _, arg := range args {
		switch {
		case arg == longFlag || strings.HasPrefix(arg, longFlag+"="):
			return true
		case shorthand != "" && (arg == shortFlag || strings.HasPrefix(arg, shortFlag+"=") || strings.HasPrefix(arg, shortFlag) && !strings.HasPrefix(arg, "--")):
			return true
		}
	}
	return false
}
