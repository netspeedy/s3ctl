---
title: Bucket policies
description: Attach a built-in template or your own policy document to each bucket s3ctl creates.
---

A bucket policy is attached to the bucket itself, unlike the policy on a
[scoped credential](/s3ctl/guides/scoped-credentials/), which is attached to a user. s3ctl applies one
after creating the bucket when you pass either a built-in template or a file. Use one or the other;
s3ctl refuses both together.

## Built-in templates

```bash
s3ctl --bucket app-data --bucket-policy-template deny-insecure-transport --dry-run
```

| Template | Effect |
| --- | --- |
| `deny-insecure-transport` | Denies every S3 action on the bucket and its objects when the request doesn't use TLS. |
| `public-read` | Allows anyone to `s3:GetObject` the bucket's objects. |

[Policy templates](/s3ctl/reference/policy-templates/) shows the documents in full.

## Your own document

```bash
s3ctl --bucket app-data --bucket-policy-file ./policies/app-data.json --dry-run
```

The file is a standard S3 bucket policy in JSON. In a config file, a relative path resolves from the
config file's directory; in a batch file, from the CSV's directory.

## On OVHcloud

Bucket policies only apply to the `s3` provider. OVHcloud controls access with container profiles
and S3 user policies instead, which s3ctl sets up on every bucket it creates there. See the
[OVHcloud overview](/s3ctl/ovhcloud/).
