---
title: IAM permissions
description: What the identity running s3ctl needs, for each thing you ask it to do.
---

s3ctl uses the credentials you give it, so the identity behind them needs permission for whatever
the run does. These are the actions behind each API call s3ctl makes on the `s3` provider.

## Creating buckets

| Action | Needed for |
| --- | --- |
| `s3:ListBucket` | Checking the bucket doesn't exist yet. s3ctl won't take over an existing bucket. |
| `s3:CreateBucket` | Every bucket. |
| `s3:PutBucketVersioning` | `--enable-versioning`. |
| `s3:PutBucketPolicy` | `--bucket-policy-template` or `--bucket-policy-file`. |

## Scoped credentials

Scoped credentials go through the IAM API as well as S3. AWS IAM is the default target; set
`--iam-endpoint` for an IAM-compatible service elsewhere.

| Action | Needed for |
| --- | --- |
| `iam:CreateUser` | The bucket's user. |
| `iam:TagUser` | The `ManagedBy` and `Bucket` tags s3ctl puts on the user. |
| `iam:PutUserPolicy` | The `s3ctl-bucket-access` inline policy. |
| `iam:CreateAccessKey` | The user's access key. |
| `iam:DeleteUserPolicy`, `iam:DeleteUser` | Cleaning up a user when a later step fails. |

## Deleting buckets

| Action | Needed for |
| --- | --- |
| `s3:ListBucket` | Checking the bucket exists and listing its objects. |
| `s3:ListBucketVersions` | Listing object versions and delete markers. |
| `s3:DeleteObject`, `s3:DeleteObjectVersion` | Emptying the bucket with `--force`. |
| `s3:DeleteBucket` | Deleting the bucket. |

## OVHcloud

On OVHcloud, s3ctl works through the Public Cloud API, so what matters is the IAM policy on your
Public Cloud project. [OAuth2 and IAM setup](/s3ctl/ovhcloud/oauth2-and-iam/#grant-it-access-to-the-project)
lists the least-privilege actions.
