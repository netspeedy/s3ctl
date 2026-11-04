---
title: JSON output and errors
description: With --output json, s3ctl writes one JSON document to stdout for successes and failures alike.
---

Text output is for people. Pass `--output json`, or set `"output": "json"` in the config, and s3ctl
writes a single JSON document to stdout instead, whether the run succeeds or fails. A failed run
still exits non-zero, so scripts can check the exit status and then read the payload rather than
scrape text.

## A successful run

This is the dry run from the [quickstart](/s3ctl/quickstart/) with `--output json`:

```json title="Output"
{
  "operation": "provision",
  "dry_run": true,
  "resource_count": 1,
  "resources": [
    {
      "bucket_name": "app-data",
      "endpoint": "https://objects.example.com",
      "region": "us-east-1",
      "created": true,
      "versioning_enabled": false,
      "encryption_enabled": false,
      "scoped_credentials": {
        "user_name": "app-data",
        "policy_template": "bucket-readwrite",
        "access_key_id": "(generated on apply)",
        "secret_access_key": "(generated on apply)"
      }
    }
  ]
}
```

| Field | Meaning |
| --- | --- |
| `operation` | `provision`, `delete` or `repair`. |
| `dry_run` | Whether this was a plan only. |
| `config_file` | The config file that was loaded, when there was one. |
| `resource_count` | How many buckets the run covered. |
| `resources` | One entry per bucket, below. |

Each entry in `resources` has `bucket_name`, `region`, `created`, `versioning_enabled` and
`encryption_enabled`, plus these when they apply:

| Field | When it appears |
| --- | --- |
| `endpoint` | The S3 endpoint the bucket lives on. |
| `scoped_credentials` | Scoped credentials were created or planned: `user_name`, `policy_template`, `access_key_id` and `secret_access_key`, with `provider` and `user_id` on OVHcloud. |
| `bucket_policy_applied`, `bucket_policy_source` | An S3 bucket policy document was applied, and the template name or file path it came from. |
| `scoped_access_policy_applied` | OVHcloud's container profile and scoped S3 user policy were applied. |
| `deleted`, `objects_deleted` | The bucket was deleted, and how many objects `--force` removed first. |
| `credentials_rotated`, `credentials_deleted` | An OVHcloud rotation issued a new key, and how many old keys a rotation or delete removed. |
| `warnings` | Something went wrong after the main work succeeded, such as an old key that couldn't be deleted. |

## A failed run

On failure the payload keeps the run's context and adds an `error` object:

```json title="Output"
{
  "operation": "delete",
  "dry_run": false,
  "config_file": "/home/operator/.config/s3ctl/config.json",
  "resource_count": 1,
  "error": {
    "code": "not_found",
    "message": "OVH bucket/container \"app-data\" does not exist in region \"UK\"; nothing was deleted",
    "detail": "OVHcloud API error ..."
  }
}
```

`message` is the same text a person would see, and `detail`, when present, carries the underlying
API error. `code` is one of:

| Code | Meaning |
| --- | --- |
| `configuration_error` | The flags, config or batch file couldn't be used. Nothing was attempted. |
| `not_found` | A bucket to delete doesn't exist, so nothing was deleted. |
| `already_exists` | A bucket to create already exists. s3ctl stops before changing it. |
| `timeout` | The run went past `--timeout`. |
| `operation_failed` | Anything else that failed while talking to the provider. |
