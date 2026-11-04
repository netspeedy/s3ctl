---
title: Bucket deletion
description: Delete OVHcloud containers along with their dedicated user and credentials.
---

OVHcloud buckets are containers, but the delete still uses the bucket name:

```bash
s3ctl --provider ovh --bucket app-data --delete --dry-run
s3ctl --provider ovh --bucket app-data --delete
```

## What happens

s3ctl looks up the container's owner and creates a temporary S3 credential for that Public Cloud
user. Through the S3-compatible API it checks whether the container is empty:

- an empty container is deleted without `--force`
- a container that still holds objects needs `--force`, which lets s3ctl empty it through the
  S3-compatible API first

The container itself is then deleted through the Public Cloud API. After that, s3ctl deletes the
user's S3 credentials and the Public Cloud user. If the container is gone but one of those clean-up
calls fails, s3ctl prints a warning so you can remove the stale identity by hand.

## Bucket-dedicated users

Deletion, [credential rotation](/s3ctl/ovhcloud/credential-rotation/) and
[policy repair](/s3ctl/ovhcloud/policy-repair/) only continue when the container's owner looks
bucket-dedicated: the Public Cloud user's description or username has to match the bucket name, or
the older description `s3ctl bucket <bucket>`. That keeps s3ctl from managing credentials or
policies on a shared user someone created by hand.
