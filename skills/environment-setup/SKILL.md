---
name: environment-setup
description: Configure and manage development, staging, production, and test environments. Use when setting up environment variables, managing configurations, or separating environments. Handles .env files, .env.test, config management, and environment-specific settings. Always use this skill when the user mentions env files, environment variables, .env.test, test environment config, or wants to separate environments — even if they just ask to "add a test env" or "configure testing variables".
metadata:
  tags: environment, configuration, env-variables, dotenv, config-management, testing
  platforms: Claude, ChatGPT, Gemini
---

# Environment Configuration

## When to use this skill

- **New Projects**: Initial environment setup
- **Multiple Environments**: Separate dev, staging, production, test
- **Team Collaboration**: Share consistent environments
- **Testing**: Isolated test environment with safe, mock-friendly values

## Instructions

### Step 1: .env File Structure

**.env.example** (template — commit this):
```bash
# Application
NODE_ENV=development
PORT=3000
APP_URL=http://localhost:3000

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/myapp
DATABASE_POOL_MIN=2
DATABASE_POOL_MAX=10

# Redis
REDIS_URL=redis://localhost:6379
REDIS_TTL=3600

# Authentication
JWT_ACCESS_SECRET=change-me-in-production-min-32-characters
JWT_REFRESH_SECRET=change-me-in-production-min-32-characters
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# External APIs
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_PUBLISHABLE_KEY=pk_test_xxx
AWS_ACCESS_KEY_ID=AKIAXXXXXXXX
AWS_SECRET_ACCESS_KEY=xxxxxxxx
AWS_REGION=us-east-1
AWS_S3_BUCKET=myapp-uploads

# Monitoring
SENTRY_DSN=https://xxx@sentry.io/xxx
LOG_LEVEL=info

# Feature Flags
ENABLE_2FA=false
ENABLE_ANALYTICS=true
```

**.env.local** (per developer — gitignore):
```bash
DATABASE_URL=postgresql://localhost:5432/myapp_dev
LOG_LEVEL=debug
```

**.env.production** (gitignore or vault):
```bash
NODE_ENV=production
PORT=8080
APP_URL=https://myapp.com

DATABASE_URL=${DATABASE_URL}
REDIS_URL=${REDIS_URL}

JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}

LOG_LEVEL=warn
ENABLE_2FA=true
```

---

### Step 2: .env.test — Test Environment

**.env.test** is loaded during test runs (`NODE_ENV=test`). It should use:
- **In-memory or separate test databases** — never the dev/prod DB
- **Fixed, deterministic secrets** — so tests are reproducible
- **Disabled or mocked external services** — email, payments, analytics
- **Fast settings** — lower timeouts, reduced pool sizes

**.env.test**:
```bash
NODE_ENV=test
PORT=3001
APP_URL=http://localhost:3001

# Isolated test database (separate from dev)
DATABASE_URL=postgresql://user:password@localhost:5432/myapp_test
DATABASE_POOL_MIN=1
DATABASE_POOL_MAX=5

# In-memory Redis for tests (or test-specific instance)
REDIS_URL=redis://localhost:6379/1
REDIS_TTL=60

# Fixed secrets — safe to commit since these are test-only
JWT_ACCESS_SECRET=test-secret-access-key-min-32-characters-x
JWT_REFRESH_SECRET=test-secret-refresh-key-min-32-characters-x
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Disable or mock external services
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=test@test.com
SMTP_PASSWORD=test

# Use Stripe test keys or a mock
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_PUBLISHABLE_KEY=pk_test_xxx

# Disable AWS calls — use local mock (e.g. localstack) or leave empty
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_REGION=us-east-1
AWS_S3_BUCKET=myapp-test-uploads

# Disable monitoring in tests
SENTRY_DSN=
LOG_LEVEL=error

# Disable features that would trigger side effects
ENABLE_2FA=false
ENABLE_ANALYTICS=false
```

**Key principles for .env.test:**
- Never share database with dev or production
- Can contain real credentials for a dedicated test database — keep it out of version control
- External service calls should be disabled or pointed to mocks
- `LOG_LEVEL=error` keeps test output clean
- `.env.test` **must NOT be committed** — add to `.gitignore`

**Loading .env.test in test runners:**

Jest (`jest.config.ts`):
```typescript
import type { Config } from 'jest';

const config: Config = {
  setupFiles: ['dotenv/config'],
  // dotenv will load .env.test when NODE_ENV=test
  // or explicitly:
  // setupFiles: ['<rootDir>/tests/loadEnv.ts'],
};
export default config;
```

Explicit loader (`tests/loadEnv.ts`):
```typescript
import dotenv from 'dotenv';
dotenv.config({ path: '.env.test' });
```

Vitest (`vitest.config.ts`):
```typescript
import { defineConfig } from 'vitest/config';
export default defineConfig({
  test: {
    env: { NODE_ENV: 'test' },
    setupFiles: ['./tests/loadEnv.ts'],
  },
});
```

---

### Step 3: Type-Safe Environment Variables (TypeScript)

**config/env.ts**:
```typescript
import { z } from 'zod';
import dotenv from 'dotenv';

dotenv.config();

const envSchema = z.object({
  NODE_ENV: z.enum(['development', 'production', 'test']),
  PORT: z.coerce.number().default(3000),
  DATABASE_URL: z.string().url(),
  JWT_ACCESS_SECRET: z.string().min(32),
  JWT_REFRESH_SECRET: z.string().min(32),
  SMTP_HOST: z.string(),
  SMTP_PORT: z.coerce.number(),
  SMTP_USER: z.string().email(),
  SMTP_PASSWORD: z.string(),
  STRIPE_SECRET_KEY: z.string().startsWith('sk_'),
  LOG_LEVEL: z.enum(['error', 'warn', 'info', 'debug']).default('info'),
});

export const env = envSchema.parse(process.env);
```

**Error Handling**:
```typescript
try {
  const env = envSchema.parse(process.env);
} catch (error) {
  if (error instanceof z.ZodError) {
    console.error('❌ Invalid environment variables:');
    error.errors.forEach((err) => {
      console.error(`  - ${err.path.join('.')}: ${err.message}`);
    });
    process.exit(1);
  }
}
```

---

### Step 4: Per-Environment Config Files

**config/environments/development.ts**:
```typescript
export default {
  logging: { level: 'debug', prettyPrint: true },
  cors: { origin: '*', credentials: true },
  rateLimit: { enabled: false },
};
```

**config/environments/production.ts**:
```typescript
export default {
  logging: { level: 'warn', prettyPrint: false },
  cors: {
    origin: process.env.ALLOWED_ORIGINS?.split(',') || [],
    credentials: true,
  },
  rateLimit: { enabled: true, windowMs: 15 * 60 * 1000, max: 100 },
};
```

**config/environments/test.ts**:
```typescript
export default {
  logging: { level: 'error', prettyPrint: false },
  cors: { origin: '*', credentials: true },
  rateLimit: { enabled: false },
  // Disable side-effecting services
  email: { disabled: true },
  analytics: { disabled: true },
};
```

**config/index.ts** (unified):
```typescript
import development from './environments/development';
import production from './environments/production';
import test from './environments/test';

const env = process.env.NODE_ENV || 'development';

const configs = { development, production, test };

export const environmentConfig = configs[env as keyof typeof configs] ?? development;
```

---

### Step 5: Docker Environment Variables

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  app:
    build: .
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgresql://postgres:password@db:5432/myapp
      - REDIS_URL=redis://redis:6379
    env_file:
      - .env.local
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: myapp

  redis:
    image: redis:7-alpine
```

**docker-compose.test.yml** (for CI):
```yaml
version: '3.8'

services:
  app-test:
    build: .
    command: npm test
    env_file:
      - .env.test
    environment:
      - NODE_ENV=test
      - DATABASE_URL=postgresql://postgres:password@db-test:5432/myapp_test
    depends_on:
      - db-test

  db-test:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: myapp_test
```

---

## Output format

```
project/
├── .env.example           # Template (commit)
├── .env                   # Local dev (gitignore)
├── .env.local             # Per developer (gitignore)
├── .env.test              # Test environment (gitignore — puede tener credenciales reales)
├── .env.production        # Production (gitignore or vault)
├── config/
│   ├── index.ts           # Main configuration
│   ├── env.ts             # Environment variable validation
│   └── environments/
│       ├── development.ts
│       ├── production.ts
│       └── test.ts
└── .gitignore
```

**.gitignore**:
```
.env
.env.local
.env.*.local
.env.test
.env.production
```

---

## Constraints

### Required Rules

1. **Provide .env.example**: Full list of required variables
2. **Validation**: Fail fast when required variables are missing
3. **.gitignore**: Never commit `.env`, `.env.local`, `.env.production`
4. **Test DB isolation**: `.env.test` must point to a separate database — never dev or production
5. **Gitignore `.env.test`**: Puede contener credenciales reales de la DB de testing

### Prohibited

1. **Commit real secrets**: Never commit `.env.production` or files with real credentials
2. **Shared test/dev DB**: Tests must never run against the development database
3. **Hardcoding**: No environment-specific values hardcoded in source

---

## Best practices

1. **12 Factor App**: All config via environment variables
2. **Type Safety**: Runtime validation with Zod
3. **Secrets Management**: Use AWS Secrets Manager, Vault, or similar for production
4. **Test isolation**: `.env.test` uses a dedicated DB and mocked/disabled external services
5. **Never commit `.env.test`**: Puede contener credenciales reales de la DB de testing — siempre en `.gitignore`

## References

- [dotenv](https://github.com/motdotla/dotenv)
- [Zod](https://zod.dev/)
- [12 Factor App - Config](https://12factor.net/config)
- [Jest env setup](https://jestjs.io/docs/configuration#setupfiles-array)
- [Vitest env setup](https://vitest.dev/config/#setupfiles)

## Metadata

### Version
- **Current Version**: 1.1.0
- **Last Updated**: 2026-03-22
- **Compatible Platforms**: Claude, ChatGPT, Gemini

### Tags
`#environment` `#configuration` `#env-variables` `#dotenv` `#config-management` `#testing` `#env-test`
