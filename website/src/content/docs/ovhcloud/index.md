---
title: OVHcloud Object Storage
description: Create OVHcloud buckets and bucket-scoped S3 keys through the Public Cloud API with --provider ovh.
sidebar:
  label: Overview
---

`--provider ovh` creates OVHcloud Object Storage through the Public Cloud API. OVHcloud calls buckets
"containers", but s3ctl keeps calling them buckets because the credentials it hands back are
S3-compatible.

For each bucket, s3ctl:

1. creates one Public Cloud user and one S3 credential pair
2. creates the container in `--region`
3. attaches the user to the container with an OVHcloud container profile, `readWrite` by default
4. imports an S3 user policy scoped to that bucket

It doesn't apply S3 bucket policy documents. On OVHcloud, access is controlled by container profiles
and S3 user policies instead.

```bash
s3ctl --provider ovh --bucket app-data --region UK \
  --ovh-service-name PUBLIC_CLOUD_PROJECT_ID --output json
```

## Required settings

| Setting | Value |
| --- | --- |
| `provider` | `ovh` |
| `ovh_service_name` | The Public Cloud project ID, the long ID shown under the project name rather than its display name. |
| `region` | A Public Cloud Object Storage region such as `UK`, `GRA`, `BHS`, `SBG` or `EU-WEST-PAR`. |
| authentication | One of: an OAuth2 service account, an access token, classic OVH API application credentials, or go-ovh's own discovery such as `ovh.conf`. |

Use the uppercase region the Public Cloud API returns. s3ctl also accepts lowercase S3 endpoint
regions such as `uk` and normalises them for the API calls.

## Optional settings

| Setting | Effect |
| --- | --- |
| `ovh_api_endpoint` | API endpoint name, such as `ovh-eu`, `ovh-ca` or `ovh-us`, or a custom API URL. |
| `ovh_client_id`, `ovh_client_secret` | OAuth2 service account credentials. |
| `ovh_access_token` | A short-lived OVHcloud access token. |
| `ovh_application_key`, `ovh_application_secret`, `ovh_consumer_key` | Classic OVH API application credentials. |
| `ovh_s3_endpoint` | Overrides the returned S3 endpoint when `https://s3.<region>.io.cloud.ovh.net` isn't right for your project. |
| `ovh_user_role` | The created user's role. Defaults to `objectstore_operator`. |
| `ovh_storage_policy_role` | `admin`, `deny`, `readOnly`, `readWrite` or `replication`. Defaults to `readWrite`. |
| `ovh_encrypt_data` | `true` turns on server-side encryption with OVH-managed keys (`AES256`, SSE-OMK). `false` asks for `plaintext` storage. |
| `ovh_tags` | Tags for new containers, such as `{"environment": "prod", "owner": "platform"}`. On the command line, repeat `--ovh-tag environment=prod`. |
| `ovh_rotate_credentials` | Rotates an existing bucket's key instead of creating a container. See [Credential rotation](/s3ctl/ovhcloud/credential-rotation/). |
| `ovh_repair_policies` | Reapplies an existing bucket's profile and policy without new keys. See [Policy repair](/s3ctl/ovhcloud/policy-repair/). |

Use `replication` only for buckets that act as replication targets. It uses OVHcloud's native
`admin` container profile plus an S3 user policy that keeps access scoped to the bucket, allows the
versioning and configuration reads and the replication target object actions OVHcloud supports, and
denies bucket administration writes.

## How the access policy behaves

The generated user policy denies `s3:ListAllMyBuckets`, so a bucket's key can't enumerate every
bucket in the project. To list a bucket's objects with MinIO's client, name the bucket:
`mc ls alias/bucket-name`. A bare `mc ls alias` uses the account-level bucket listing, which OVHcloud
documents as all buckets or denied rather than a filtered list.

For `readOnly` and `readWrite`, s3ctl also adds explicit denies for unsupported operations on the
owned bucket. OVHcloud falls back to the bucket owner's ACL when a user policy has neither an allow
nor a deny for an action, so the explicit denies are what keep an owner-scoped user in its lane.

JSON output reports this container profile and user policy as `scoped_access_policy_applied`.
`bucket_policy_applied` only appears when an S3 bucket policy document was actually applied.
