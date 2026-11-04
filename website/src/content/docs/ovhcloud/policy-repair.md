---
title: Policy repair
description: Reapply the scoped access policies on existing OVHcloud buckets without touching their keys.
---

Use `--ovh-repair-policies` when the buckets and keys already exist and you only need their scoped
access policies applied again:

```bash
s3ctl \
  --provider ovh \
  --bucket netspeedy-archives \
  --ovh-repair-policies \
  --output json
```

Pass several `--bucket` values, or a [batch file](/s3ctl/guides/batch-provisioning/), to repair many
bucket users in one run. For each bucket, s3ctl finds the container's `ownerId`, checks the owner
still looks bucket-dedicated, reapplies the OVHcloud container profile and imports a freshly
generated S3 user policy. It doesn't create, delete or rotate any keys.

## Widening one bucket for replication

To let a single bucket act as a replication target without changing any others, repair just that
bucket with the `replication` profile:

```bash
s3ctl \
  --provider ovh \
  --bucket netspeedy-archives \
  --ovh-storage-policy-role replication \
  --ovh-repair-policies \
  --output json
```

## After a leak

If a key may have been copied somewhere it shouldn't be, repair the policy and then
[rotate](/s3ctl/ovhcloud/credential-rotation/), so the old key stops working:

```bash
s3ctl \
  --provider ovh \
  --bucket netspeedy-archives \
  --ovh-rotate-credentials \
  --output json
```

`--ovh-repair-policies` can't be combined with `--delete` or `--ovh-rotate-credentials`.
