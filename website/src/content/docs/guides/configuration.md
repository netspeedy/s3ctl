---
title: Configuration
description: Keep shared provider settings in a JSON config so each run only names what's specific to it.
---

Settings resolve in this order:

1. flags on the command line
2. the JSON config file
3. built-in defaults

So shared provider settings can live in the config while anything specific to one run stays on
the command line. Most flags have a config key of the same name in snake case: `--enable-versioning`
becomes `enable_versioning`. The [CLI reference](/s3ctl/reference/cli/) notes the exceptions.

## A config file

```json title="examples/aws/s3ctl.json"
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

Run it with `--config`:

```bash
s3ctl --config ./examples/aws/s3ctl.json --dry-run --output json
```

Relative paths inside a config file resolve from the config file's own directory, so a batch file
or a policy document can sit beside it.

## Your default config

Without `--config`, s3ctl loads the first of these that exists:

- `$XDG_CONFIG_HOME/s3ctl/config.json`
- `$HOME/.config/s3ctl/config.json`

That's the place for settings every run shares: the provider, endpoint, region, profile or
credentials, IAM and OVHcloud defaults, and the output format.

```json title="~/.config/s3ctl/config.json"
{
  "endpoint": "https://objects.example.com",
  "region": "us-east-1",
  "access_key": "MASTER_ACCESS_KEY_ID",
  "secret_key": "MASTER_SECRET_ACCESS_KEY",
  "create_scoped_credentials": true,
  "credential_policy_template": "bucket-readwrite"
}
```

With that in place, this is enough to plan a bucket and its key:

```bash
s3ctl --bucket app-data --dry-run
```

Install the example as your default with permissions only you can read:

```bash
install -d -m 700 "${XDG_CONFIG_HOME:-$HOME/.config}/s3ctl"
install -m 600 ./examples/aws/user-config.json "${XDG_CONFIG_HOME:-$HOME/.config}/s3ctl/config.json"
```

## Credentials

Use either `profile` or an explicit `access_key` and `secret_key`, not both; s3ctl refuses the
combination. Add `session_token` when the credentials are temporary. When none of them are set, the
AWS SDK's normal credential and profile discovery applies, the same as the AWS CLI.

:::caution
If the default config holds secrets, keep it out of any repository and readable only by you.
:::

## OVHcloud

An OVHcloud config names the provider, the Public Cloud project and one way to authenticate. With an
OAuth2 service account:

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

Classic OVH API application credentials work too:

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

[OAuth2 and IAM setup](/s3ctl/ovhcloud/oauth2-and-iam/) covers creating either set of credentials.
