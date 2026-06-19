package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func provision(ctx context.Context, cfg settings) (provisionResult, error) {
	targets, err := buildProvisionTargets(cfg)
	if err != nil {
		return provisionResult{}, err
	}

	result := provisionResult{
		Operation:     operationProvision,
		DryRun:        cfg.DryRun,
		ConfigFile:    cfg.ConfigPath,
		ResourceCount: len(targets),
		Resources:     make([]resourceResult, 0, len(targets)),
	}
	if cfg.DeleteBucket {
		result.Operation = operationDelete
	} else if cfg.OVHRepairPolicies {
		result.Operation = operationRepair
	}

	if cfg.Provider == providerOVH {
		return provisionWithOVH(ctx, cfg, targets, result)
	}

	var s3Client s3API
	var iamClient iamAPI

	if !cfg.DryRun {
		s3Client, err = newS3APIClient(ctx, cfg)
		if err != nil {
			return provisionResult{}, err
		}
	}

	if cfg.DeleteBucket {
		return deleteS3Buckets(ctx, cfg, targets, result, s3Client)
	}

	for _, target := range targets {
		resource := resourceResult{
			BucketName: target.Bucket,
			Endpoint:   cfg.Endpoint,
			Region:     cfg.Region,
		}

		bucketPolicyDocument, bucketPolicySource, err := resolveBucketPolicy(target)
		if err != nil {
			return provisionResult{}, err
		}

		if cfg.DryRun {
			resource.Created = true
			resource.VersioningEnabled = target.EnableVersioning
			resource.BucketPolicyApplied = bucketPolicyDocument != ""
			resource.BucketPolicySource = bucketPolicySource

			if target.CreateScopedCredentials {
				userName, err := resolvedIAMUserName(target, cfg.IAMUserPrefix)
				if err != nil {
					return provisionResult{}, err
				}
				resource.ScopedCredentials = &scopedCredentialResult{
					UserName:        userName,
					PolicyTemplate:  target.CredentialPolicyTemplate,
					AccessKeyID:     "(generated on apply)",
					SecretAccessKey: "(generated on apply)",
				}
			}

			result.Resources = append(result.Resources, resource)
			continue
		}

		exists, err := bucketExists(ctx, s3Client, target.Bucket)
		if err != nil {
			return provisionResult{}, err
		}
		if exists {
			return provisionResult{}, bucketExistsError{Name: target.Bucket}
		}

		if err := createBucket(ctx, s3Client, target.Bucket, cfg.Region); err != nil {
			return provisionResult{}, err
		}
		resource.Created = true

		if target.EnableVersioning {
			if err := enableVersioning(ctx, s3Client, target.Bucket); err != nil {
				return provisionResult{}, err
			}
			resource.VersioningEnabled = true
		}

		if bucketPolicyDocument != "" {
			if err := applyBucketPolicy(ctx, s3Client, target.Bucket, bucketPolicyDocument); err != nil {
				return provisionResult{}, err
			}
			resource.BucketPolicyApplied = true
			resource.BucketPolicySource = bucketPolicySource
		}

		if target.CreateScopedCredentials {
			if iamClient == nil {
				iamClient, err = newIAMClient(ctx, cfg)
				if err != nil {
					return provisionResult{}, err
				}
			}

			credentials, err := createScopedCredentials(ctx, iamClient, target, cfg)
			if err != nil {
				return provisionResult{}, err
			}
			resource.ScopedCredentials = &credentials
		}

		result.Resources = append(result.Resources, resource)
	}

	return result, nil
}

func buildProvisionTargets(cfg settings) ([]provisionTarget, error) {
	targets := make([]provisionTarget, 0, len(cfg.Buckets))
	for _, bucket := range dedupeStringsPreserveOrder(cfg.Buckets) {
		if strings.TrimSpace(bucket) == "" {
			continue
		}

		targets = append(targets, provisionTarget{
			Bucket:                   bucket,
			EnableVersioning:         cfg.EnableVersioning,
			BucketPolicyFile:         cfg.BucketPolicyFile,
			BucketPolicyTemplate:     cfg.BucketPolicyTemplate,
			CreateScopedCredentials:  cfg.CreateScopedCredentials,
			IAMUserName:              cfg.IAMUserName,
			CredentialPolicyTemplate: cfg.CredentialPolicyTemplate,
		})
	}

	if cfg.BatchFile != "" {
		batchTargets, err := loadBatchTargets(cfg.BatchFile, cfg)
		if err != nil {
			return nil, err
		}
		targets = append(targets, batchTargets...)
	}

	if len(targets) == 0 {
		return nil, errors.New("no bucket targets were resolved from flags, config, or batch file")
	}

	if cfg.IAMUserName != "" && len(targets) > 1 {
		return nil, errors.New("--iam-user-name can only be used when provisioning a single bucket")
	}

	seenBuckets := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if _, exists := seenBuckets[target.Bucket]; exists {
			return nil, fmt.Errorf("bucket target %q was specified more than once; each bucket must only appear once per run", target.Bucket)
		}
		seenBuckets[target.Bucket] = struct{}{}
	}

	return targets, nil
}
