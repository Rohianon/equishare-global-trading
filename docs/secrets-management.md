# Secrets Management Strategy

This document outlines how EquiShare manages configuration and secrets across different environments.

## Overview

EquiShare uses a layered configuration approach:

```
┌─────────────────────────────────────────────────────────────┐
│ Environment Variables (EQUISHARE_*)         ← Highest      │
├─────────────────────────────────────────────────────────────┤
│ .env.local (personal overrides)                             │
├─────────────────────────────────────────────────────────────┤
│ .env (shared team defaults)                                 │
├─────────────────────────────────────────────────────────────┤
│ config.yaml (checked into git)                              │
├─────────────────────────────────────────────────────────────┤
│ Default values in code                      ← Lowest        │
└─────────────────────────────────────────────────────────────┘
```

## Configuration by Environment

### Local Development

For local development, use `.env` files:

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your local values

3. For personal overrides that shouldn't affect the team, use `.env.local`

**Files:**
- `.env` - Shared team defaults (git-ignored)
- `.env.local` - Personal overrides (git-ignored)
- `.env.example` - Template (checked into git)

### Staging Environment

Staging uses the same configuration as production but with test credentials:

- Secrets stored in AWS SSM Parameter Store
- Injected as environment variables by the deployment pipeline
- Separate parameter paths: `/equishare/staging/*`

### Production Environment

Production secrets are managed through AWS SSM Parameter Store:

```
/equishare/production/database/password
/equishare/production/jwt/secret
/equishare/production/mpesa/consumer_key
/equishare/production/mpesa/consumer_secret
...
```

**Access Model:**
- ECS tasks have IAM roles with SSM read access
- Secrets are injected at container startup
- No secrets stored in container images or code

## Secret Categories

### Critical Secrets (Rotate Regularly)
| Secret | Rotation Period | Notes |
|--------|----------------|-------|
| JWT Secret | 90 days | Invalidates all sessions |
| Database Password | 90 days | Coordinate with RDS |
| M-Pesa API Keys | 180 days | Coordinate with Safaricom |
| Alpaca API Keys | 90 days | Paper vs Live separation |

### Standard Secrets
| Secret | Storage | Notes |
|--------|---------|-------|
| Redis Password | SSM | May be empty in dev |
| Kafka SASL | SSM | Only for secured clusters |
| KYC API Key | SSM | Provider-specific |
| SMS API Key | SSM | Africa's Talking or Twilio |

## Loading Configuration in Code

```go
package main

import "github.com/Rohianon/equishare-global-trading/pkg/config"

func main() {
    // Basic loading (development)
    cfg := config.MustLoad("config")

    // With validation (production)
    cfg := config.MustLoadWithValidation("config", config.Requirements{
        Database: true,
        JWT:      true,
        MPesa:    true,
    })

    // Check environment
    if config.IsProduction() {
        // Production-specific initialization
    }
}
```

## AWS SSM Parameter Store Setup

### Creating Parameters

```bash
# Create a secret
aws ssm put-parameter \
    --name "/equishare/production/jwt/secret" \
    --value "your-secret-value" \
    --type "SecureString" \
    --key-id "alias/equishare-secrets"

# List parameters
aws ssm get-parameters-by-path \
    --path "/equishare/production" \
    --recursive
```

### IAM Policy for ECS

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ssm:GetParameters",
                "ssm:GetParameter"
            ],
            "Resource": "arn:aws:ssm:*:*:parameter/equishare/production/*"
        },
        {
            "Effect": "Allow",
            "Action": "kms:Decrypt",
            "Resource": "arn:aws:kms:*:*:key/YOUR-KMS-KEY-ID"
        }
    ]
}
```

### ECS Task Definition

```json
{
    "containerDefinitions": [
        {
            "name": "api-gateway",
            "secrets": [
                {
                    "name": "EQUISHARE_JWT_SECRET",
                    "valueFrom": "arn:aws:ssm:us-east-1:123456789:parameter/equishare/production/jwt/secret"
                },
                {
                    "name": "EQUISHARE_DATABASE_PASSWORD",
                    "valueFrom": "arn:aws:ssm:us-east-1:123456789:parameter/equishare/production/database/password"
                }
            ]
        }
    ]
}
```

## Validation

Services validate required configuration at startup:

```go
cfg, err := config.LoadWithValidation("config", config.Requirements{
    Database:  true,  // Requires database credentials
    JWT:       true,  // Requires JWT secret (32+ chars)
    MPesa:     true,  // Requires M-Pesa credentials
    Telemetry: true,  // Requires telemetry config if enabled
})
if err != nil {
    log.Fatalf("Configuration error: %v", err)
}
```

Failed validation produces clear error messages:

```
configuration validation failed:
  - database.user: required
  - database.password: required
  - jwt.secret: must be at least 32 characters
```

## Security Best Practices

1. **Never commit secrets** - All `.env` files are in `.gitignore`

2. **Use strong secrets** - Generate with:
   ```bash
   openssl rand -base64 32
   ```

3. **Separate environments** - Production secrets should never be used in development

4. **Rotate regularly** - Follow the rotation schedule above

5. **Audit access** - Use CloudTrail to monitor SSM access

6. **Encrypt at rest** - Use KMS for SSM SecureString parameters

7. **Least privilege** - Each service only accesses its required secrets

## Troubleshooting

### Service fails to start with config error

1. Check environment variables: `env | grep EQUISHARE`
2. Verify SSM permissions (in AWS)
3. Check parameter paths match expectations

### Secrets not loading in ECS

1. Verify IAM role has SSM access
2. Check parameter ARNs in task definition
3. Review ECS task logs for SSM errors

### Local development issues

1. Ensure `.env` exists: `cp .env.example .env`
2. Check file permissions
3. Verify no syntax errors in `.env`
