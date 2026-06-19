package cli

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func loadBatchTargets(path string, cfg settings) ([]provisionTarget, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("batch file %s is empty", path)
		}
		return nil, err
	}

	headerIndex := make(map[string]int, len(headers))
	for index, header := range headers {
		headerIndex[normalizeCSVHeader(header)] = index
	}

	if !hasCSVHeader(headerIndex, "bucket", "bucket_name", "name") {
		return nil, fmt.Errorf("batch file %s must include a bucket column", path)
	}

	batchDir := filepath.Dir(path)
	targets := make([]provisionTarget, 0)
	lineNumber := 1

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		lineNumber++

		if csvRecordBlank(record) || csvRecordComment(record) {
			continue
		}

		bucket := csvField(record, headerIndex, "bucket", "bucket_name", "name")
		if bucket == "" {
			return nil, fmt.Errorf("batch file %s line %d is missing a bucket value", path, lineNumber)
		}

		target := provisionTarget{
			Bucket:                   bucket,
			EnableVersioning:         cfg.EnableVersioning,
			BucketPolicyFile:         cfg.BucketPolicyFile,
			BucketPolicyTemplate:     cfg.BucketPolicyTemplate,
			CreateScopedCredentials:  cfg.CreateScopedCredentials,
			IAMUserName:              cfg.IAMUserName,
			CredentialPolicyTemplate: cfg.CredentialPolicyTemplate,
		}

		if value := csvField(record, headerIndex, "iam_user_name", "iam_user", "user_name"); value != "" {
			target.IAMUserName = value
		}
		if value := csvField(record, headerIndex, "bucket_policy_file"); value != "" {
			target.BucketPolicyFile = resolveRelativePath(batchDir, value)
			target.BucketPolicyTemplate = ""
		}
		if value := csvField(record, headerIndex, "bucket_policy_template"); value != "" {
			target.BucketPolicyTemplate = value
			target.BucketPolicyFile = ""
		}
		if value := csvField(record, headerIndex, "credential_policy_template", "iam_policy_template"); value != "" {
			target.CredentialPolicyTemplate = value
		}
		if value := csvField(record, headerIndex, "enable_versioning", "versioning"); value != "" {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("batch file %s line %d has invalid enable_versioning value %q", path, lineNumber, value)
			}
			target.EnableVersioning = parsed
		}
		if value := csvField(record, headerIndex, "create_scoped_credentials", "create_credentials", "create_iam_user"); value != "" {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("batch file %s line %d has invalid create_scoped_credentials value %q", path, lineNumber, value)
			}
			target.CreateScopedCredentials = parsed
		}

		if target.BucketPolicyTemplate != "" {
			if _, ok := bucketPolicyTemplates[target.BucketPolicyTemplate]; !ok {
				return nil, fmt.Errorf("batch file %s line %d uses unsupported bucket policy template %q", path, lineNumber, target.BucketPolicyTemplate)
			}
		}
		if target.CredentialPolicyTemplate != "" {
			if _, ok := credentialPolicyTemplates[target.CredentialPolicyTemplate]; !ok {
				return nil, fmt.Errorf("batch file %s line %d uses unsupported credential policy template %q", path, lineNumber, target.CredentialPolicyTemplate)
			}
		}

		targets = append(targets, target)
	}

	return targets, nil
}

func normalizeCSVHeader(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(value)
}

func hasCSVHeader(index map[string]int, aliases ...string) bool {
	for _, alias := range aliases {
		if _, ok := index[alias]; ok {
			return true
		}
	}
	return false
}

func csvField(record []string, index map[string]int, aliases ...string) string {
	for _, alias := range aliases {
		column, ok := index[alias]
		if !ok || column >= len(record) {
			continue
		}
		value := strings.TrimSpace(record[column])
		if value != "" {
			return value
		}
	}
	return ""
}

func csvRecordBlank(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func csvRecordComment(record []string) bool {
	for _, value := range record {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		return strings.HasPrefix(trimmed, "#")
	}
	return false
}

func resolveRelativePath(baseDir, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func resolveRelativePathIfSet(baseDir, value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	return resolveRelativePath(baseDir, value)
}
