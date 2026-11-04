---
title: Policy templates
description: The exact policy documents behind each bucket policy and scoped credential template.
---

There are two sets. Bucket policy templates attach to the bucket itself and only apply to the `s3`
provider. Credential templates attach to the user that `--create-scoped-credentials` creates. Every
example here is for a bucket called `app-data`; s3ctl substitutes the real bucket name.

## Credential templates

Chosen with `--credential-policy-template`, or `credential_policy_template` in a config or batch
file. On the `s3` provider each is attached to the scoped user as an inline policy named
`s3ctl-bucket-access`.

### bucket-readonly

Find the bucket's location, list it and read objects.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BucketReadOnlyBucketMeta",
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": ["arn:aws:s3:::app-data"]
    },
    {
      "Sid": "BucketReadOnlyObjects",
      "Effect": "Allow",
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::app-data/*"]
    }
  ]
}
```

### bucket-readwrite

The default. Everything `bucket-readonly` allows, plus writes, deletes and multipart uploads.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BucketReadWriteBucketMeta",
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket", "s3:ListBucketMultipartUploads"],
      "Resource": ["arn:aws:s3:::app-data"]
    },
    {
      "Sid": "BucketReadWriteObjects",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:AbortMultipartUpload",
        "s3:ListMultipartUploadParts"
      ],
      "Resource": ["arn:aws:s3:::app-data/*"]
    }
  ]
}
```

### bucket-admin

Every S3 action on the bucket and its objects.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BucketAdminAccess",
      "Effect": "Allow",
      "Action": ["s3:*"],
      "Resource": ["arn:aws:s3:::app-data", "arn:aws:s3:::app-data/*"]
    }
  ]
}
```

## Bucket policy templates

Chosen with `--bucket-policy-template`. Use `--bucket-policy-file` instead for a document of your own.

### deny-insecure-transport

Denies every S3 action on the bucket and its objects when the request doesn't use TLS.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyInsecureTransport",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": ["arn:aws:s3:::app-data", "arn:aws:s3:::app-data/*"],
      "Condition": { "Bool": { "aws:SecureTransport": "false" } }
    }
  ]
}
```

### public-read

Lets anyone read the bucket's objects. Only use it for content that's meant to be public.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadObjects",
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::app-data/*"]
    }
  ]
}
```

## On OVHcloud

OVHcloud doesn't use these documents. It scopes each bucket's user with a container profile and an
S3 user policy that s3ctl generates, chosen by `--ovh-storage-policy-role`. See the
[OVHcloud overview](/s3ctl/ovhcloud/).
