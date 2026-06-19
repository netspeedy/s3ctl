package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

var bucketNotFoundPattern = regexp.MustCompile(`\b(NotFound|NoSuchBucket)\b`)

var bucketPolicyTemplates = map[string]string{
	"public-read":             "Allow public read access to objects in the bucket",
	"deny-insecure-transport": "Deny requests that do not use TLS",
}

type s3API interface {
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	CreateBucket(context.Context, *s3.CreateBucketInput, ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	PutBucketVersioning(context.Context, *s3.PutBucketVersioningInput, ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error)
	PutBucketPolicy(context.Context, *s3.PutBucketPolicyInput, ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error)
	ListObjectVersions(context.Context, *s3.ListObjectVersionsInput, ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObjects(context.Context, *s3.DeleteObjectsInput, ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	DeleteBucket(context.Context, *s3.DeleteBucketInput, ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

var newS3APIClient = func(ctx context.Context, cfg settings) (s3API, error) {
	return newS3Client(ctx, cfg)
}

func newAWSConfig(ctx context.Context, cfg settings) (aws.Config, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.Profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(cfg.Profile))
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		loadOptions = append(
			loadOptions,
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
			),
		)
	}

	if cfg.Insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		loadOptions = append(loadOptions, awsconfig.WithHTTPClient(&http.Client{Transport: transport}))
	}

	return awsconfig.LoadDefaultConfig(ctx, loadOptions...)
}

func newS3Client(ctx context.Context, cfg settings) (*s3.Client, error) {
	awsCfg, err := newAWSConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = true
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	}), nil
}

func bucketExists(ctx context.Context, client s3API, bucket string) (bool, error) {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return true, nil
	}

	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == http.StatusNotFound {
		return false, nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchBucket":
			return false, nil
		}
	}

	if bucketNotFoundPattern.MatchString(err.Error()) {
		return false, nil
	}

	return false, fmt.Errorf("unable to determine whether bucket exists: %w", err)
}

func createBucket(ctx context.Context, client s3API, bucket, region string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}
	if region != defaultRegion {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	if _, err := client.CreateBucket(ctx, input); err != nil {
		return fmt.Errorf("failed to create bucket %q: %w", bucket, err)
	}

	return nil
}

func enableVersioning(ctx context.Context, client s3API, bucket string) error {
	_, err := client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to enable versioning on bucket %q: %w", bucket, err)
	}
	return nil
}

func applyBucketPolicy(ctx context.Context, client s3API, bucket, policy string) error {
	_, err := client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	if err != nil {
		return fmt.Errorf("failed to apply bucket policy to %q: %w", bucket, err)
	}
	return nil
}

func deleteS3Buckets(ctx context.Context, cfg settings, targets []provisionTarget, result provisionResult, client s3API) (provisionResult, error) {
	for _, target := range targets {
		resource := resourceResult{
			BucketName: target.Bucket,
			Endpoint:   cfg.Endpoint,
			Region:     cfg.Region,
			Deleted:    true,
		}

		if cfg.DryRun {
			result.Resources = append(result.Resources, resource)
			continue
		}

		exists, err := bucketExists(ctx, client, target.Bucket)
		if err != nil {
			return provisionResult{}, err
		}
		if !exists {
			return provisionResult{}, bucketNotFoundError{Name: target.Bucket, Provider: providerS3}
		}

		if cfg.Force {
			deleted, err := emptyS3Bucket(ctx, client, target.Bucket)
			if err != nil {
				return provisionResult{}, err
			}
			resource.ObjectsDeleted = deleted
		} else {
			if err := ensureS3BucketEmpty(ctx, client, target.Bucket); err != nil {
				return provisionResult{}, err
			}
		}

		if err := deleteS3Bucket(ctx, client, target.Bucket); err != nil {
			return provisionResult{}, err
		}

		result.Resources = append(result.Resources, resource)
	}

	return result, nil
}

func ensureS3BucketEmpty(ctx context.Context, client s3API, bucket string) error {
	hasObjectVersions, err := s3BucketHasObjectVersions(ctx, client, bucket)
	if err != nil {
		return err
	}
	if hasObjectVersions {
		return fmt.Errorf("refusing to delete non-empty bucket %q without --force; rerun with --delete --force to remove objects, versions, and delete markers before deleting the bucket", bucket)
	}

	hasCurrentObjects, err := s3BucketHasCurrentObjects(ctx, client, bucket)
	if err != nil {
		return err
	}
	if hasCurrentObjects {
		return fmt.Errorf("refusing to delete non-empty bucket %q without --force; rerun with --delete --force to remove objects before deleting the bucket", bucket)
	}

	return nil
}

func emptyS3Bucket(ctx context.Context, client s3API, bucket string) (int, error) {
	versionedDeleted, err := deleteS3ObjectVersions(ctx, client, bucket)
	if err != nil {
		return 0, err
	}

	currentDeleted, err := deleteS3CurrentObjects(ctx, client, bucket)
	if err != nil {
		return 0, err
	}

	return versionedDeleted + currentDeleted, nil
}

func s3BucketHasObjectVersions(ctx context.Context, client s3API, bucket string) (bool, error) {
	input := &s3.ListObjectVersionsInput{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	}

	for {
		output, err := client.ListObjectVersions(ctx, input)
		if err != nil {
			if isUnsupportedObjectVersionListing(err) {
				return false, nil
			}
			return false, fmt.Errorf("failed to list object versions in bucket %q: %w", bucket, err)
		}

		for _, version := range output.Versions {
			if version.Key != nil {
				return true, nil
			}
		}
		for _, marker := range output.DeleteMarkers {
			if marker.Key != nil {
				return true, nil
			}
		}

		if !aws.ToBool(output.IsTruncated) {
			return false, nil
		}
		if output.NextKeyMarker == nil && output.NextVersionIdMarker == nil {
			return false, fmt.Errorf("failed to continue listing object versions in bucket %q: truncated response did not include a next marker", bucket)
		}
		input.KeyMarker = output.NextKeyMarker
		input.VersionIdMarker = output.NextVersionIdMarker
	}
}

func s3BucketHasCurrentObjects(ctx context.Context, client s3API, bucket string) (bool, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	}

	for {
		output, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return false, fmt.Errorf("failed to list current objects in bucket %q: %w", bucket, err)
		}

		for _, object := range output.Contents {
			if object.Key != nil {
				return true, nil
			}
		}

		if !aws.ToBool(output.IsTruncated) {
			return false, nil
		}
		if output.NextContinuationToken == nil {
			return false, fmt.Errorf("failed to continue listing current objects in bucket %q: truncated response did not include a continuation token", bucket)
		}
		input.ContinuationToken = output.NextContinuationToken
	}
}

func deleteS3ObjectVersions(ctx context.Context, client s3API, bucket string) (int, error) {
	input := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	deleted := 0
	for {
		output, err := client.ListObjectVersions(ctx, input)
		if err != nil {
			if isUnsupportedObjectVersionListing(err) {
				return deleted, nil
			}
			return deleted, fmt.Errorf("failed to list object versions in bucket %q: %w", bucket, err)
		}

		objects := make([]types.ObjectIdentifier, 0, len(output.Versions)+len(output.DeleteMarkers))
		for _, version := range output.Versions {
			if version.Key == nil {
				continue
			}
			objects = append(objects, types.ObjectIdentifier{
				Key:       version.Key,
				VersionId: version.VersionId,
			})
		}
		for _, marker := range output.DeleteMarkers {
			if marker.Key == nil {
				continue
			}
			objects = append(objects, types.ObjectIdentifier{
				Key:       marker.Key,
				VersionId: marker.VersionId,
			})
		}

		count, err := deleteS3ObjectBatch(ctx, client, bucket, objects)
		if err != nil {
			return deleted, err
		}
		deleted += count

		if !aws.ToBool(output.IsTruncated) {
			return deleted, nil
		}
		if output.NextKeyMarker == nil && output.NextVersionIdMarker == nil {
			return deleted, fmt.Errorf("failed to continue listing object versions in bucket %q: truncated response did not include a next marker", bucket)
		}
		input.KeyMarker = output.NextKeyMarker
		input.VersionIdMarker = output.NextVersionIdMarker
	}
}

func deleteS3CurrentObjects(ctx context.Context, client s3API, bucket string) (int, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	deleted := 0
	for {
		output, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return deleted, fmt.Errorf("failed to list current objects in bucket %q: %w", bucket, err)
		}

		objects := make([]types.ObjectIdentifier, 0, len(output.Contents))
		for _, object := range output.Contents {
			if object.Key == nil {
				continue
			}
			objects = append(objects, types.ObjectIdentifier{Key: object.Key})
		}

		count, err := deleteS3ObjectBatch(ctx, client, bucket, objects)
		if err != nil {
			return deleted, err
		}
		deleted += count

		if !aws.ToBool(output.IsTruncated) {
			return deleted, nil
		}
		if output.NextContinuationToken == nil {
			return deleted, fmt.Errorf("failed to continue listing current objects in bucket %q: truncated response did not include a continuation token", bucket)
		}
		input.ContinuationToken = output.NextContinuationToken
	}
}

func deleteS3ObjectBatch(ctx context.Context, client s3API, bucket string, objects []types.ObjectIdentifier) (int, error) {
	deleted := 0
	for len(objects) > 0 {
		batchSize := min(len(objects), s3DeleteObjectsBatchSize)
		batch := objects[:batchSize]
		objects = objects[batchSize:]

		output, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{
				Objects: batch,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return deleted, fmt.Errorf("failed to delete objects from bucket %q: %w", bucket, err)
		}
		if len(output.Errors) > 0 {
			return deleted, fmt.Errorf("failed to delete %d object(s) from bucket %q: %s", len(output.Errors), bucket, renderS3DeleteErrors(output.Errors))
		}
		deleted += len(batch)
	}
	return deleted, nil
}

func deleteS3Bucket(ctx context.Context, client s3API, bucket string) error {
	if _, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}); err != nil {
		return fmt.Errorf("failed to delete bucket %q: %w", bucket, err)
	}
	return nil
}

func renderS3DeleteErrors(deleteErrors []types.Error) string {
	parts := make([]string, 0, min(len(deleteErrors), 3))
	for _, deleteErr := range deleteErrors {
		key := aws.ToString(deleteErr.Key)
		code := aws.ToString(deleteErr.Code)
		message := aws.ToString(deleteErr.Message)
		switch {
		case key != "" && code != "" && message != "":
			parts = append(parts, fmt.Sprintf("%s (%s: %s)", key, code, message))
		case key != "" && code != "":
			parts = append(parts, fmt.Sprintf("%s (%s)", key, code))
		case key != "":
			parts = append(parts, key)
		case code != "":
			parts = append(parts, code)
		default:
			parts = append(parts, "unknown delete error")
		}
		if len(parts) == 3 {
			break
		}
	}
	if len(deleteErrors) > len(parts) {
		parts = append(parts, fmt.Sprintf("and %d more", len(deleteErrors)-len(parts)))
	}
	return strings.Join(parts, "; ")
}

func isUnsupportedObjectVersionListing(err error) bool {
	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) {
		switch responseErr.HTTPStatusCode() {
		case http.StatusMethodNotAllowed, http.StatusNotImplemented:
			return true
		}
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "MethodNotAllowed", "NotImplemented", "NotSupported", "XNotImplemented":
			return true
		}
	}

	return false
}

func isS3AccessDenied(err error) bool {
	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) {
		switch responseErr.HTTPStatusCode() {
		case http.StatusUnauthorized, http.StatusForbidden:
			return true
		}
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AccessDenied", "AllAccessDisabled", "InvalidAccessKeyId", "InvalidToken", "SignatureDoesNotMatch":
			return true
		}
	}

	return false
}

func resolveBucketPolicy(target provisionTarget) (string, string, error) {
	if target.BucketPolicyFile != "" {
		data, err := os.ReadFile(target.BucketPolicyFile)
		if err != nil {
			return "", "", err
		}
		if !json.Valid(data) {
			return "", "", fmt.Errorf("bucket policy file is not valid JSON: %s", target.BucketPolicyFile)
		}
		return string(data), target.BucketPolicyFile, nil
	}

	if target.BucketPolicyTemplate != "" {
		document, err := buildBucketPolicy(target.Bucket, target.BucketPolicyTemplate)
		if err != nil {
			return "", "", err
		}
		return document, target.BucketPolicyTemplate, nil
	}

	return "", "", nil
}

func buildBucketPolicy(bucket, template string) (string, error) {
	bucketARN := fmt.Sprintf("arn:aws:s3:::%s", bucket)
	objectARN := bucketARN + "/*"

	var document map[string]any
	switch template {
	case "public-read":
		document = map[string]any{
			"Version": "2012-10-17",
			"Statement": []map[string]any{
				{
					"Sid":       "PublicReadObjects",
					"Effect":    "Allow",
					"Principal": "*",
					"Action":    []string{"s3:GetObject"},
					"Resource":  []string{objectARN},
				},
			},
		}
	case "deny-insecure-transport":
		document = map[string]any{
			"Version": "2012-10-17",
			"Statement": []map[string]any{
				{
					"Sid":       "DenyInsecureTransport",
					"Effect":    "Deny",
					"Principal": "*",
					"Action":    "s3:*",
					"Resource":  []string{bucketARN, objectARN},
					"Condition": map[string]any{
						"Bool": map[string]string{
							"aws:SecureTransport": "false",
						},
					},
				},
			},
		}
	default:
		return "", fmt.Errorf("unsupported bucket policy template %q", template)
	}

	bytes, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
