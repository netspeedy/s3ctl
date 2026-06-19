package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func writeJSONError(w io.Writer, cfg settings, err error, fallbackCode string) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(commandErrorResult{
		Operation:     operationFromSettings(cfg),
		DryRun:        cfg.DryRun,
		ConfigFile:    cfg.ConfigPath,
		ResourceCount: len(cfg.Buckets),
		Error: commandErrorDetail{
			Code:    errorCode(err, fallbackCode),
			Message: renderErrorMessage(err),
			Detail:  renderErrorDetail(err),
		},
	})
}

func renderErrorMessage(err error) string {
	var notFound bucketNotFoundError
	if errors.As(err, &notFound) {
		return notFound.Error()
	}
	return err.Error()
}

func renderErrorDetail(err error) string {
	var notFound bucketNotFoundError
	if errors.As(err, &notFound) && notFound.Cause != nil {
		return notFound.Cause.Error()
	}
	return ""
}

func errorCode(err error, fallback string) string {
	var notFound bucketNotFoundError
	if errors.As(err, &notFound) {
		return "not_found"
	}
	var exists bucketExistsError
	if errors.As(err, &exists) {
		return "already_exists"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if fallback != "" {
		return fallback
	}
	return "error"
}

func renderText(result provisionResult) string {
	title := "S3 Provisioning Result"
	switch result.Operation {
	case operationDelete:
		title = "S3 Delete Result"
	case operationRepair:
		title = "S3 Policy Repair Result"
	}
	lines := []string{
		title,
		strings.Repeat("=", len(title)),
		fmt.Sprintf("Resources: %d", result.ResourceCount),
	}

	if result.ConfigFile != "" {
		lines = append(lines, fmt.Sprintf("Config file: %s", result.ConfigFile))
	}
	if result.DryRun {
		lines = append(lines, "Mode: dry-run")
	}

	for _, resource := range result.Resources {
		if result.Operation == operationRepair {
			policyRepairLabel := "Scoped access policy repaired"
			if result.DryRun {
				policyRepairLabel = "Scoped access policy repair planned"
			}
			lines = append(lines,
				"",
				fmt.Sprintf("Bucket: %s", resource.BucketName),
				fmt.Sprintf("Endpoint: %s", emptyFallback(resource.Endpoint, "(default AWS SDK resolution)")),
				fmt.Sprintf("Region: %s", resource.Region),
				fmt.Sprintf("%s: %s", policyRepairLabel, yesNo(resource.AccessPolicyApplied)),
			)
			for _, warning := range resource.Warnings {
				lines = append(lines, fmt.Sprintf("Warning: %s", warning))
			}
			continue
		}

		if result.Operation == operationDelete {
			bucketDeleteLabel := "Bucket deleted"
			if result.DryRun {
				bucketDeleteLabel = "Bucket delete planned"
			}
			lines = append(lines,
				"",
				fmt.Sprintf("Bucket: %s", resource.BucketName),
				fmt.Sprintf("Endpoint: %s", emptyFallback(resource.Endpoint, "(default AWS SDK resolution)")),
				fmt.Sprintf("Region: %s", resource.Region),
				fmt.Sprintf("%s: %s", bucketDeleteLabel, yesNo(resource.Deleted)),
			)
			if !result.DryRun {
				lines = append(lines, fmt.Sprintf("Objects deleted: %d", resource.ObjectsDeleted))
				if resource.CredentialsDeleted > 0 {
					lines = append(lines, fmt.Sprintf("Credentials deleted: %d", resource.CredentialsDeleted))
				}
			}
			for _, warning := range resource.Warnings {
				lines = append(lines, fmt.Sprintf("Warning: %s", warning))
			}
			continue
		}

		bucketCreateLabel := "Bucket created"
		versioningLabel := "Versioning enabled"
		encryptionLabel := "Encryption enabled"
		bucketPolicyLabel := "Bucket policy applied"
		scopedCredentialLabel := "Scoped credentials created"
		if result.DryRun {
			bucketCreateLabel = "Bucket create planned"
			versioningLabel = "Versioning requested"
			encryptionLabel = "Encryption requested"
			bucketPolicyLabel = "Bucket policy planned"
			scopedCredentialLabel = "Scoped credentials planned"
		}

		lines = append(lines,
			"",
			fmt.Sprintf("Bucket: %s", resource.BucketName),
			fmt.Sprintf("Endpoint: %s", emptyFallback(resource.Endpoint, "(default AWS SDK resolution)")),
			fmt.Sprintf("Region: %s", resource.Region),
			fmt.Sprintf("%s: %s", bucketCreateLabel, yesNo(resource.Created)),
			fmt.Sprintf("%s: %s", versioningLabel, yesNo(resource.VersioningEnabled)),
			fmt.Sprintf("%s: %s", encryptionLabel, yesNo(resource.EncryptionEnabled)),
			fmt.Sprintf("%s: %s", bucketPolicyLabel, yesNo(resource.BucketPolicyApplied)),
		)

		if resource.BucketPolicySource != "" {
			lines = append(lines, fmt.Sprintf("Bucket policy source: %s", resource.BucketPolicySource))
		}

		if resource.ScopedCredentials != nil {
			identityLabel := "IAM user"
			policyLabel := "Credential policy template"
			if resource.ScopedCredentials.Provider == providerOVH {
				identityLabel = "OVH user"
				policyLabel = "OVH storage policy role"
			}

			lines = append(lines,
				fmt.Sprintf("%s: %s", scopedCredentialLabel, yesNo(true)),
				fmt.Sprintf("%s: %s", identityLabel, resource.ScopedCredentials.UserName),
				fmt.Sprintf("%s: %s", policyLabel, resource.ScopedCredentials.PolicyTemplate),
				fmt.Sprintf("Access key ID: %s", resource.ScopedCredentials.AccessKeyID),
				fmt.Sprintf("Secret access key: %s", resource.ScopedCredentials.SecretAccessKey),
			)
			if resource.ScopedCredentials.UserID != "" {
				lines = append(lines, fmt.Sprintf("User ID: %s", resource.ScopedCredentials.UserID))
			}
		}

		if resource.CredentialsRotated {
			rotationLabel := "Credentials rotated"
			if result.DryRun {
				rotationLabel = "Credential rotation planned"
			}
			lines = append(lines, fmt.Sprintf("%s: yes", rotationLabel))
			if !result.DryRun {
				lines = append(lines, fmt.Sprintf("Previous credentials deleted: %d", resource.CredentialsDeleted))
			}
		}
		for _, warning := range resource.Warnings {
			lines = append(lines, fmt.Sprintf("Warning: %s", warning))
		}
	}

	return strings.Join(lines, "\n")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
