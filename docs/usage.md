# Usage Guide

`s3ctl --help` is a short operator quick reference. Use `s3ctl --help-full`
when you need the complete flag, template, and batch CSV reference.

## Common commands

Plan a single bucket with generated scoped credentials:

```bash
s3ctl \
  --bucket app-data \
  --endpoint https://objects.example.com \
  --region us-east-1 \
  --create-scoped-credentials \
  --credential-policy-template bucket-readwrite \
  --dry-run
```

Provision an OVHcloud Object Storage container and a dedicated S3 key:

```bash
s3ctl \
  --provider ovh \
  --bucket app-data \
  --region UK \
  --ovh-service-name PUBLIC_CLOUD_PROJECT_ID \
  --output json
```

Rotate an existing OVHcloud bucket keypair:

```bash
s3ctl \
  --provider ovh \
  --bucket app-data \
  --ovh-rotate-credentials \
  --output json
```

Repair OVHcloud bucket scoping for an existing bucket user:

```bash
s3ctl \
  --provider ovh \
  --bucket app-data \
  --ovh-repair-policies \
  --output json
```

Show focused bucket workflow help:

```bash
s3ctl --bucket app-data --help
```

Show the full CLI reference:

```bash
s3ctl --help-full
```

## Batch provisioning

For bulk runs, the normal pattern is:

1. Define the shared provider settings once with flags or config.
2. Feed the bucket list in with repeated `--bucket` flags or `--batch-file`.
3. Let `s3ctl` generate one scoped user and one access key pair per bucket.

Plan multiple buckets from repeated flags:

```bash
s3ctl \
  --bucket app-data \
  --bucket logs-archive \
  --create-scoped-credentials \
  --dry-run \
  --output json
```

Plan a batch from CSV:

```bash
s3ctl \
  --batch-file ./examples/aws/s3ctl-batch.csv \
  --endpoint https://objects.example.com \
  --region us-east-1 \
  --create-scoped-credentials \
  --dry-run \
  --output json
```

Supported batch CSV columns:

- `bucket`
- `iam_user_name`
- `enable_versioning`
- `bucket_policy_file`
- `bucket_policy_template`
- `create_scoped_credentials`
- `credential_policy_template`

Example CSV:

```csv
bucket,create_scoped_credentials,credential_policy_template,enable_versioning
app-data,true,bucket-readwrite,true
logs-archive,true,bucket-readonly,false
```

## Configuration

Configuration is resolved in this order:

1. CLI flags
2. JSON config file
3. Built-in defaults

Example config:

```json
{
  "endpoint": "https://objects.example.com",
  "region": "us-east-1",
  "enable_versioning": true,
  "create_scoped_credentials": true,
  "credential_policy_template": "bucket-readwrite",
  "bucket_policy_template": "deny-insecure-transport",
  "batch_file": "./s3ctl-batch.csv"
}
```

Run it:

```bash
s3ctl --config ./examples/aws/s3ctl.json --dry-run --output json
```

When `--output json` or `"output": "json"` is set, command failures are also
written to stdout as JSON. The process still exits non-zero, but automation can
read the `error.code`, `error.message`, and optional `error.detail` fields
instead of scraping text:

```json
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

Example OVHcloud config with OAuth2 service account credentials:

```json
{
  "provider": "ovh",
  "ovh_service_name": "PUBLIC_CLOUD_PROJECT_ID",
  "ovh_client_id": "CLIENT_ID",
  "ovh_client_secret": "CLIENT_SECRET",
  "region": "UK",
  "enable_versioning": true,
  "ovh_encrypt_data": true,
  "ovh_storage_policy_role": "readWrite",
  "output": "json"
}
```

Classic OVH API application credentials are also supported:

```json
{
  "provider": "ovh",
  "ovh_service_name": "PROJECT_ID",
  "ovh_application_key": "APPLICATION_KEY",
  "ovh_application_secret": "APPLICATION_SECRET",
  "ovh_consumer_key": "CONSUMER_KEY",
  "region": "GRA"
}
```

With that saved in your default config, this is enough:

```bash
s3ctl --bucket app-data
```

Relative paths inside the config file are resolved from the config file
directory, so config-local batch files and policy documents work cleanly.

Default user config path:

- `$XDG_CONFIG_HOME/s3ctl/config.json`
- `$HOME/.config/s3ctl/config.json`

When `--config` is unset, `s3ctl` will automatically load that default file if
it exists. This is the right place for shared operator settings such as
provider, endpoint, region, profile, credentials, IAM/OVH defaults, and output
preferences.

Example default user config:

```json
{
  "endpoint": "https://objects.example.com",
  "region": "us-east-1",
  "access_key": "MASTER_ACCESS_KEY_ID",
  "secret_key": "MASTER_SECRET_ACCESS_KEY",
  "create_scoped_credentials": true,
  "credential_policy_template": "bucket-readwrite"
}
```

Use either `profile` or explicit `access_key` and `secret_key` values, not both.
Add `session_token` when your master credentials are temporary. If those values
are not set in `s3ctl`, the AWS SDK still uses its normal credential and profile
discovery. If you keep secrets in the default user config, store that file
outside the repository and restrict its permissions.

Install that as your per-user default:

```bash
install -d -m 700 "${XDG_CONFIG_HOME:-$HOME/.config}/s3ctl"
install -m 600 ./examples/aws/user-config.json "${XDG_CONFIG_HOME:-$HOME/.config}/s3ctl/config.json"
```

## Built-in templates

Bucket policy templates:

| Template | Coverage |
| --- | --- |
| `deny-insecure-transport` | Denies all S3 actions against the bucket and objects when requests do not use secure transport. |
| `public-read` | Allows public `s3:GetObject` access to objects in the bucket. |

Scoped credential policy templates:

| Template | Coverage |
| --- | --- |
| `bucket-readonly` | Allows bucket location lookup, bucket listing, and object reads for one bucket. |
| `bucket-readwrite` | Allows bucket location lookup, bucket listing, object reads, writes, deletes, and multipart upload operations for one bucket. |
| `bucket-admin` | Allows all S3 actions against one bucket and its objects. |

By default, generated scoped credentials use `bucket-readwrite`, generated IAM
user names are derived directly from bucket names, and no IAM path is set.
Configure `iam_user_prefix` or `--iam-user-prefix` when generated user names
should share a prefix. Configure `iam_path` or `--iam-path` when generated users
should be created under an IAM path.

## IAM notes

Scoped credential provisioning uses the IAM API in addition to the S3 API. The
principal running `s3ctl` therefore needs permission to:

- create buckets and apply bucket configuration in S3
- create IAM users
- attach inline IAM policies
- create IAM access keys

AWS IAM is the default target. When you need an IAM-compatible alternative, use
`--iam-endpoint` or `iam_endpoint` in JSON config.

## Deleting buckets

Use `--delete` with one or more `--bucket` values to remove buckets instead of
creating them. Empty buckets can be deleted without `--force`. Non-empty buckets
require `--force`; without it, `s3ctl` lists the bucket contents and refuses to
delete the bucket if objects, object versions, or delete markers are present.
Use `--dry-run` to preview the target.

```bash
s3ctl --bucket app-data --delete --dry-run
s3ctl --bucket app-data --delete
s3ctl --bucket app-data --delete --force --timeout 30m
```

Without `--force`, the S3 provider only lists object versions, delete markers,
and current objects to confirm the bucket is empty before deleting it. With
`--force`, it deletes object versions and delete markers when the endpoint
supports version listing, deletes any remaining current objects, and finally
deletes the bucket.

The S3 principal running the delete needs the matching list, object delete,
object version delete, and bucket delete permissions.

JSON config can also drive this mode:

```json
{
  "bucket": "app-data",
  "delete_bucket": true
}
```

The shorter `"delete": true` config key is accepted as an alias for
`"delete_bucket": true`.

Keep `"force": true` out of shared default configs unless every run using that
config should be allowed to remove bucket contents before deleting buckets.

Use `--timeout` or `"timeout": "30m"` for large buckets or slower
object-storage endpoints. The default timeout is `10m`.
