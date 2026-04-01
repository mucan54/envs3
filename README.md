# envs3

Encrypted environment variable management over S3-compatible storage.

envs3 is a CLI tool that manages encrypted `.env` files stored on any S3-compatible backend (Cloudflare R2, AWS S3, MinIO, DigitalOcean Spaces). Secrets are encrypted client-side using envelope encryption — the storage backend only ever sees ciphertext.

## Why envs3?

Teams share `.env` files through insecure channels: Slack messages, shared password managers, wikis, or tribal knowledge. Existing solutions either require a centralized server where the provider can see your secrets, or commit encrypted files to git which creates merge conflicts.

envs3 provides:

- **Single-command sync** — `envs3 pull` updates your `.env`. That's the entire daily workflow.
- **User-owned storage** — Secrets live in your own S3 bucket, not on third-party servers.
- **Zero-knowledge** — No server, service, or storage backend ever sees plaintext secrets.
- **Framework-agnostic** — envs3 writes a standard `.env` file. Every framework reads it natively.

## Quick Start

### 1. Initialize a project (admin)

```bash
envs3 init
```

This walks you through an interactive setup:
- Choose your storage backend (R2, S3, MinIO, etc.)
- Enter your S3 credentials (read-write for admin, read-only for team)
- Name your project and environments
- Optionally import an existing `.env` file

The command generates your keypair, creates the project in your bucket, and writes `.envs3.json` to your project root.

### 2. Commit the config

```bash
git add .envs3.json
git commit -m "Add envs3 config"
```

`.envs3.json` is safe to commit — it only contains read-only S3 credentials. The encrypted data in your bucket is undecipherable without a user's private key.

### 3. Daily workflow

```bash
# Fetch latest secrets
envs3 pull

# Make changes locally, then push
envs3 push

# Or modify remote environments directly
envs3 set DB_HOST=newhost.com --env=production
```

### 4. Onboard a team member

**New member** runs:
```bash
git clone <repo>              # Gets .envs3.json
envs3 auth setup              # Generates keypair
envs3 pubkey --output my.pub  # Exports public key
# Send my.pub to admin
```

**Admin** runs:
```bash
envs3 members add my.pub --env=local,staging
```

**New member** can now:
```bash
envs3 pull   # Decrypts and writes .env
```

### 5. CI/CD

**Admin** creates a token:
```bash
envs3 token create --env=production --name=github-deploy
# Outputs: ENVS3_TOKEN=envs3_v1_eyJ...
```

**In CI** (GitHub Actions, etc.):
```yaml
steps:
  - name: Fetch secrets
    env:
      ENVS3_TOKEN: ${{ secrets.ENVS3_TOKEN }}
    run: envs3 pull
```

## Installation

### From source

```bash
git clone https://github.com/mucan54/envs3.git
cd envs3
make build
# Binary at ./bin/envs3
```

### Go install

```bash
go install github.com/mucan54/envs3/cmd/envs3@latest
```

## Commands

### Core Workflow

| Command | Description |
|---------|-------------|
| `envs3 init` | Initialize a new project (interactive) |
| `envs3 pull [env]` | Fetch and decrypt secrets to `.env` |
| `envs3 push` | Push local `.env` changes to the active environment |
| `envs3 status` | Show current project state |

### Remote Operations

These commands operate on remote state **without touching your local `.env`**:

| Command | Description |
|---------|-------------|
| `envs3 set KEY=VALUE [--env=ENV]` | Set key-value pairs on a remote environment |
| `envs3 unset KEY [--env=ENV]` | Remove keys from a remote environment |
| `envs3 view [--env=ENV]` | Display secrets without modifying files |
| `envs3 diff [env1] [env2]` | Compare environments or local vs remote |

### History

| Command | Description |
|---------|-------------|
| `envs3 log [env]` | Show version history |
| `envs3 rollback [env] <version>` | Restore a previous version |

### Team Management

| Command | Description |
|---------|-------------|
| `envs3 members list` | List all members and their access |
| `envs3 members add <pubkey> --env=ENV` | Add a member with environment access |
| `envs3 members remove <email> [--env=ENV]` | Remove a member (triggers DEK rotation) |
| `envs3 auth setup` | Generate a new keypair |
| `envs3 pubkey` | Display your public key |

### Environments

| Command | Description |
|---------|-------------|
| `envs3 env list` | List environments |
| `envs3 env create <name> [--from=ENV]` | Create a new environment |

### Service Tokens (CI/CD)

| Command | Description |
|---------|-------------|
| `envs3 token create --env=ENV --name=NAME` | Create a service token |
| `envs3 token list` | List all tokens |
| `envs3 token revoke <name>` | Revoke a token (triggers DEK rotation) |

### Other

| Command | Description |
|---------|-------------|
| `envs3 connect` | Connect to an existing project |

### Global Flags

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help |
| `--version`, `-v` | Print version |
| `--verbose` | Enable verbose output |
| `--json` | Output in JSON format |
| `--project`, `-p` | Override project directory |

### Environment Variable Overrides

| Variable | Description |
|----------|-------------|
| `ENVS3_TOKEN` | Service token for CI/CD (overrides keypair auth) |
| `ENVS3_WRITE_KEY_ID` | Override write S3 access key ID |
| `ENVS3_WRITE_SECRET_KEY` | Override write S3 secret key |

## Command Details

### `envs3 pull`

The only command that writes to your local `.env` file. All other commands operate on remote state in memory.

```bash
envs3 pull              # Pull default environment
envs3 pull staging      # Pull a specific environment
envs3 pull --force      # Skip ETag cache, always pull
envs3 pull --output .env.local  # Write to custom path
```

When switching environments, the current `.env` is backed up to `.env.previous`.

Pull is optimized: it sends a lightweight HEAD request first. If the remote hasn't changed since your last pull (ETag match), it exits immediately without downloading anything.

### `envs3 push`

Always pushes to the **active environment** (the one you last pulled). This prevents accidentally pushing local debug values to the wrong environment.

```bash
envs3 push          # Shows diff and asks for confirmation
envs3 push --yes    # Skip confirmation
```

If someone else pushed changes since your last pull, you'll get a conflict error. Run `envs3 pull` first, then try again.

### `envs3 set` / `envs3 unset`

Modify secrets directly on a remote environment without changing your local `.env`. This is how you update production secrets while your `.env` contains local development values.

```bash
# Set keys on the active environment
envs3 set CACHE_TTL=3600

# Set keys on a specific environment
envs3 set DB_HOST=prod-db.com DB_PORT=5433 --env=production

# Set and unset in one operation
envs3 set NEW_KEY=value --unset=OLD_KEY,DEPRECATED --env=production

# Remove keys
envs3 unset OLD_KEY --env=production
```

**Confirmation safety**: Non-default environments (like production) require you to type the environment name to confirm. This prevents accidental modifications.

### `envs3 view`

View secrets without modifying any files — "pull to screen."

```bash
# View all secrets
envs3 view --env=production

# View only key names (no DEK access needed)
envs3 view --env=production --keys

# Get a single value (useful for scripting)
DB_HOST=$(envs3 view --key=DB_HOST --env=production)

# JSON output
envs3 view --env=production --format=json
```

### `envs3 diff`

```bash
envs3 diff                    # Local .env vs remote active env
envs3 diff production         # Local .env vs production
envs3 diff local production   # Compare two remote environments
envs3 diff --all              # Include identical keys
```

### `envs3 members remove`

When a member is removed, envs3 automatically **rotates the DEK** for every affected environment. This means:

1. A new encryption key is generated
2. All secrets are re-encrypted with the new key
3. The new key is sealed for remaining members only
4. The removed member's old key can no longer decrypt anything

```bash
# Remove from specific environments
envs3 members remove user@email.com --env=production

# Remove from all environments and the project
envs3 members remove user@email.com
```

## Architecture

### Access Model

envs3 uses two independent security layers:

**Layer 1 — Storage Access (S3 IAM):**

| Key Type | Permissions | Who Holds It |
|----------|------------|--------------|
| Read-Only | GetObject, HeadObject, ListBucket | All team members (in `.envs3.json`) |
| Read-Write | GetObject, HeadObject, ListBucket, PutObject, DeleteObject | Admin(s) only (in `~/.envs3/credentials/`) |

**Layer 2 — Cryptographic Access (Envelope Encryption):**

Even with storage access, a user can only decrypt secrets for environments where their public key has been added to the keyring.

### Cryptography

| Component | Algorithm |
|-----------|-----------|
| User Keypair | X25519 (Curve25519 ECDH) |
| DEK Wrapping | NaCl `crypto_box_seal` (X25519 + XSalsa20-Poly1305) |
| Secret Encryption | AES-256-GCM (12-byte nonce, 16-byte auth tag) |
| Checksums | SHA-256 |
| Encoding | Base64 (standard, with padding) |

### Envelope Encryption Flow

```
                     ┌──────────────┐
                     │  User's X25519│
                     │  Private Key  │
                     └──────┬───────┘
                            │ unwrap
                            ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  keyring.json │───►│     DEK      │───►│  current.json │
│  (wrapped DEK │    │  (AES-256    │    │  (encrypted   │
│   per user)   │    │   key)       │    │   values)     │
└──────────────┘    └──────────────┘    └──────────────┘
```

1. Each environment has a **DEK** (Data Encryption Key) — a 32-byte AES-256 key
2. The DEK is **wrapped** (encrypted) individually for each authorized user using their X25519 public key via NaCl sealed boxes
3. Secret **values** are encrypted with AES-256-GCM using the DEK
4. Secret **key names** are stored in plaintext (knowing `DB_PASSWORD` exists reveals nothing about its value)

### Storage Layout

```
s3://<bucket>/
└── <project>/
    ├── project.json          # Project metadata
    ├── members.json          # Member public keys + token metadata
    └── environments/
        └── <env>/
            ├── current.json  # Encrypted secrets (latest)
            ├── keyring.json  # DEK wrapped per authorized user
            └── history/
                ├── 000001.json
                ├── 000002.json
                └── ...
```

### Local Files

```
~/.envs3/
├── keys/
│   └── <project>.key          # X25519 private key (32 bytes, mode 0600)
├── state/
│   └── <project>.json         # ETag cache, active environment
└── credentials/
    └── <project>.json         # Admin write S3 credentials (mode 0600)
```

### Code Structure

```
cmd/envs3/main.go              # Entry point

internal/
├── format/                    # Data types + .env parsing
│   ├── project.go             # project.json types
│   ├── members.go             # members.json types
│   ├── keyring.go             # keyring.json types
│   ├── bundle.go              # current.json types
│   ├── history.go             # history entry types
│   ├── config.go              # .envs3.json + local state types
│   ├── token.go               # Service token payload types
│   └── dotenv.go              # .env file parser/writer + canonical JSON
│
├── crypto/                    # Cryptographic operations
│   ├── keypair.go             # X25519 key generation + fingerprints
│   ├── seal.go                # NaCl sealed box (wrap/unwrap DEK)
│   ├── aes.go                 # AES-256-GCM encrypt/decrypt
│   └── checksum.go            # SHA-256 checksums
│
├── config/                    # Configuration + file management
│   ├── paths.go               # ~/.envs3/ path resolution
│   ├── loader.go              # Config file loading/saving
│   └── token.go               # Service token encode/decode
│
├── storage/                   # S3 abstraction
│   ├── interface.go           # Store interface
│   ├── s3.go                  # S3 implementation (aws-sdk-go-v2)
│   ├── paths.go               # S3 key construction helpers
│   └── memory.go              # In-memory mock for testing
│
├── engine/                    # Business logic
│   ├── engine.go              # Core helpers (decrypt, encrypt, checksum)
│   ├── init.go                # Project initialization
│   ├── pull.go                # Pull with ETag optimization
│   ├── push.go                # Push with conflict detection
│   ├── set.go                 # Remote-only set/unset
│   ├── view.go                # View without file writes
│   ├── diff.go                # Environment comparison
│   ├── log.go                 # Version history
│   ├── rollback.go            # Version restore
│   ├── status.go              # Project status
│   ├── members.go             # Member add/remove + DEK rotation
│   ├── token.go               # Service token create/revoke
│   └── env.go                 # Environment create/list
│
└── cli/                       # Cobra commands (thin wrappers over engine)
    ├── root.go, init.go, pull.go, push.go, set.go, unset.go,
    ├── view.go, diff.go, log.go, rollback.go, status.go,
    ├── connect.go, env.go, members.go, token.go, auth.go, pubkey.go
    └── helpers.go
```

The architecture is strictly layered bottom-up:

```
format, crypto          (zero internal deps)
       ↓
    config              (depends on format)
       ↓
    storage             (depends on format)
       ↓
    engine              (depends on all above)
       ↓
      cli               (depends on engine + config)
```

### Conflict Detection

envs3 uses S3 ETag-based optimistic concurrency. When pushing, it sends an `If-Match` header with the ETag from the last read. If someone else modified the file in between, S3 returns `412 Precondition Failed` and envs3 asks you to pull first.

### Pull Optimization

Pulls start with a lightweight HEAD request (~50ms) to check if the ETag has changed since the last pull. If it hasn't, envs3 prints "Already up to date" without downloading any data.

## Security Model

### What envs3 protects against

| Threat | Protection |
|--------|-----------|
| S3 bucket exposed publicly | Values encrypted with AES-256-GCM |
| Read-only S3 key leaked | No DEK access = can't decrypt |
| Storage backend operator | Zero-knowledge (client-side encryption) |
| Man-in-the-middle | TLS + GCM authentication |

### What envs3 does NOT protect against

- A member who had legitimate access copying secrets while they had access (no system can prevent this — rotate credentials when members leave)
- Key names are visible (deliberate trade-off for usability)
- Denial of service via S3 (enable bucket versioning to protect against deletion)

### Recommended Practices

1. Enable S3 bucket versioning
2. Private key files are automatically set to `chmod 600`
3. Rotate CI/CD tokens periodically
4. When a member leaves, DEK rotation is automatic, but also rotate the actual credentials (database passwords, API keys) at the service level
5. Keep read-write S3 access to as few people as possible (ideally 1-2 admins)

## .gitignore

Add these to your `.gitignore`:

```
.env
.env.previous
```

**Do NOT** gitignore `.envs3.json` — it's the project config and is safe to commit.

## Comparison

| Feature | dotenvx | Infisical | Doppler | **envs3** |
|---------|---------|-----------|---------|-----------|
| `pull` writes .env directly | No | No | No | **Yes** |
| Data ownership | Git repo | Platform | Platform | **Your S3 bucket** |
| Encryption | E2E (ECIES) | Server-side | Server-side | **E2E (X25519 + AES-256-GCM)** |
| Server required | No | Yes | Yes (SaaS) | **No** |
| S3-compatible storage | No | No | No | **Yes** |
| Free tier | Yes | Limited | Limited | **Unlimited (own storage)** |
| CI/CD tokens | No | Yes | Yes | **Yes** |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint
```

### Running Tests

```bash
# All tests
go test ./... -v

# Specific package
go test ./internal/crypto/... -v
go test ./internal/engine/... -v
```

## License

MIT
