---
title: Deleting buckets
description: Delete empty buckets with --delete, and only empty a bucket when you pass --force.
---

`--delete` turns a run around: instead of creating the named buckets, s3ctl deletes them. Preview
it first with `--dry-run`.

```bash
s3ctl --bucket app-data --delete --dry-run
s3ctl --bucket app-data --delete
s3ctl --bucket app-data --delete --force --timeout 30m
```

## Empty buckets only, unless you say otherwise

Without `--force`, s3ctl lists the bucket's current objects, object versions and delete markers, and
refuses to delete it if any are there. An empty bucket is deleted straight away.

With `--force`, it deletes object versions and delete markers where the endpoint supports listing
versions, then any remaining current objects, and finally the bucket. A large bucket can take a
while, so raise `--timeout` from its 10 minute default.

The principal running the delete needs list, object delete, object version delete and bucket
delete permissions. [IAM permissions](/s3ctl/reference/iam-permissions/) has the exact actions.

## From a config file

```json
{
  "bucket": "app-data",
  "delete_bucket": true
}
```

The config key is `delete_bucket`, not `delete`.

## On OVHcloud

OVHcloud deletes also remove the bucket's dedicated user and its credentials, and only go ahead when
that user looks bucket-dedicated. See [Bucket deletion](/s3ctl/ovhcloud/bucket-deletion/).
