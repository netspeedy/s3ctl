---
title: Batch provisioning
description: Plan and create many buckets in one run, from repeated flags or a CSV file, with a scoped key for each.
---

For a bulk run the pattern is:

1. Set the shared provider settings once, with flags or a [config file](/s3ctl/guides/configuration/).
2. Name the buckets with repeated `--bucket` flags or a CSV file in `--batch-file`.
3. Let s3ctl create one scoped user and one access key per bucket.

## Repeated flags

```bash
s3ctl \
  --bucket app-data \
  --bucket logs-archive \
  --create-scoped-credentials \
  --dry-run \
  --output json
```

## A CSV file

```csv title="examples/aws/s3ctl-batch.csv"
bucket,create_scoped_credentials,credential_policy_template,enable_versioning
app-data,true,bucket-readwrite,true
logs-archive,true,bucket-readonly,false
```

```bash
s3ctl \
  --batch-file ./examples/aws/s3ctl-batch.csv \
  --endpoint https://objects.example.com \
  --region us-east-1 \
  --create-scoped-credentials \
  --dry-run \
  --output json
```

Each row starts from the shared settings, and a value in any other column overrides them for that
row. Blank lines and lines that start with `#` are skipped. A relative `bucket_policy_file` resolves
from the CSV file's directory.

## Columns

Only `bucket` is required. Headers are case-insensitive, and spaces or hyphens in them count as
underscores.

| Column | Also accepted | Value |
| --- | --- | --- |
| `bucket` | `bucket_name`, `name` | The bucket name. |
| `iam_user_name` | `iam_user`, `user_name` | The scoped user's name, instead of one derived from the bucket. |
| `enable_versioning` | `versioning` | `true` or `false`. |
| `bucket_policy_file` | | Path to a JSON bucket policy document. Replaces any template. |
| `bucket_policy_template` | | A [bucket policy template](/s3ctl/reference/policy-templates/). Replaces any file. |
| `create_scoped_credentials` | `create_credentials`, `create_iam_user` | `true` or `false`. |
| `credential_policy_template` | `iam_policy_template` | A [credential policy template](/s3ctl/reference/policy-templates/). |

A row that names an unknown template, or a value that isn't a boolean where one is expected, stops
the run before anything is created and names the line.
