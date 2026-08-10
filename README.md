# DotConfig
A config center for backend system, support access with http API and web ui to manage your configurations

## Develop and Run
install air
```shell
go install github.com/air-verse/air@latest
```
create the configuration and update the content
```shell
cp .app.toml.example .app.toml
vim .app.toml
```
then run project created by this template
```shell
air serve
```

## Lint code and commit
lint the code
```shell
golangci-lint run
```

If it has some issues, try auto fix

```shell
golangci-lint run --fix
```

If it has issues about struct alignment, try this command to fix

```shell
fieldalignment -fix ./path/to/package
```

If the `fieldalignment` command not found, you can install it with this command:
```shell
go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
```

After code style fixed, you can commit the code
```shell
git add .
git commit -m "commit_message"
```

## Build to release
```shell
cd storage/ui
npm ci
npm run build
cd ../..
GOOS=linux GOARCH=amd64 go build
```

## Configuration Center

The built-in configuration center is disabled by default. Enable it and set
the mandatory Basic Auth credentials in `.app.toml` before using the management
page or any configuration-center API:

```toml
[configcenter]
enabled=true
username="config-admin"
password="replace-with-at-least-16-random-characters"
```

The username must be 1-128 characters, without surrounding whitespace or a
colon. The password must be 16-256 characters. Startup fails if the feature is
enabled with invalid credentials. When disabled, `/config-center` and all
routes below it return `404`. Basic Auth protects the management page and
management APIs; always expose them through HTTPS because Basic Auth encodes
credentials but does not encrypt them.

The management UI is available at `/config-center`. It supports visual editing
for objects, arrays, strings, booleans, and JSON numbers. Environments can inherit
from one parent environment; objects merge recursively, while arrays and scalar
values replace the inherited value.

Configuration edits are saved as drafts. Use the multi-environment publish
action to create immutable, fully resolved snapshots in one database
transaction. Apply the Atlas migrations before first use:

```shell
atlas migrate apply --env local
```

### Configuration encryption

Draft configurations, draft descriptions, published configurations,
published descriptions, and namespace read/write API Keys support
application-layer envelope encryption. The
external keyring must not be stored in this repository, in the configuration
center database, or in a configuration namespace. The application config only
contains the provider, active key ID, and absolute keyring path:

```toml
[configcrypt]
enabled=true
provider="file"
activeKey="config-key-v1"
keyring="/run/secrets/config-center/keys.json"
```

Generate the first AES-256 key without printing the key material to the
terminal. The command creates or updates the keyring with `0600` permissions:

```shell
go run . config-key generate --config .app.toml --id config-key-v1
```

After enabling encryption on a database that already contains configuration
data, encrypt legacy plaintext records:

```shell
go run . config-key reencrypt --config .app.toml
```

To rotate the key-encryption key, generate a new key ID in the same external
keyring, update `activeKey`, restart the service, and run `reencrypt` again:

```shell
go run . config-key generate --config .app.toml --id config-key-v2
# Update activeKey to config-key-v2 and restart the service.
go run . config-key reencrypt --config .app.toml
```

The re-encryption command rewraps stored data-encryption keys with the active
key and can be safely rerun. Keep old keys in the external keyring until all
records are migrated and backups encrypted under those keys have expired. A
lost key cannot be recovered from the database.

### Published configuration API

Applications can read the latest published, fully resolved configuration by
namespace and environment slug. Draft values are never returned by this API:

```text
GET /config-center/api/runtime/{namespace}/{environment}?format=json
```

Supported formats are `json` (default), `yaml`, `toml`, `env`, and `ini`.
JSON uses the normal API response envelope and exposes configuration under
`data.config` with descriptions under `data.descriptions`. Text formats return
the configuration directly and include descriptions as comments.

Every namespace has independent read and write API Keys. The read Key is
accepted only through `X-API-Key` and can only call `GET`. The write Key is
accepted only through `X-Write-API-Key` and can only call `PATCH`; it cannot be
used to read configuration. Neither Key is returned by the namespace API, and
both are stored with the same external-keyring envelope encryption used by
configuration payloads, but with distinct encryption contexts. Creating a
namespace requires both Keys to contain 32-256 URL-safe characters. When
creating or rotating credentials, the read and write Keys must differ. When
editing a namespace, leave either field empty to keep that Key or enter a new
value to rotate it independently. Existing namespaces migrated from an older
version deny write access until a write Key has been explicitly configured.
The runtime APIs do not require the management Basic Auth credentials, but they
are available only while `configcenter.enabled=true`.

Open the management API with Basic Auth, for example:

```shell
curl --user 'config-admin:replace-with-the-config-center-password' \
  'https://config.example.com/config-center/api/namespaces'
```

```shell
curl -H 'Accept: application/json' \
  -H 'X-API-Key: replace-with-the-namespace-api-key' \
  'http://127.0.0.1:8080/config-center/api/runtime/my-service/production'

curl -H 'X-API-Key: replace-with-the-namespace-api-key' \
  'http://127.0.0.1:8080/config-center/api/runtime/my-service/production?format=yaml'
```

Responses include `X-Config-Namespace`, `X-Config-Environment`,
`X-Config-Version`, `X-Config-Batch`, and `Last-Modified` metadata headers and
use `Cache-Control: no-store` because published configuration may contain
sensitive values.

Applications and deployment automation can replace one existing configuration
value with an authenticated `PATCH` request to the same runtime resource. The
`path` is an [RFC 6901 JSON Pointer](https://www.rfc-editor.org/rfc/rfc6901), so
nested object keys and array indexes are unambiguous. The update is written as
an environment draft override and published as a new immutable version in one
database transaction; the next `GET` immediately returns the new value.

```shell
curl --fail-with-body --request PATCH \
  -H 'Content-Type: application/json' \
  -H 'X-Write-API-Key: replace-with-the-namespace-write-api-key' \
  --data '{"path":"/service/database/host","value":"db.internal"}' \
  'http://127.0.0.1:8080/config-center/api/runtime/my-service/production'
```

The target path must already exist in the fully resolved environment
configuration. This prevents a misspelled path from silently creating a new
setting. Values may be objects, arrays, strings, booleans, or JSON numbers;
`null` is rejected consistently with management-page configuration validation.

### Loading published configuration into Viper

The application can merge the latest published configuration for one
environment into Viper from the root command before any child command runs.
This makes the same configuration available to `serve`, `queue`, migration,
and other commands. `.app.toml` remains the bootstrap source and selects either
a trusted direct database read or the runtime HTTP API. Draft configuration is
never loaded. With the database source, `config-key generate` intentionally
skips remote loading because it may be creating the first keyring required to
decrypt that source; the HTTP source remains available to the command.

For a direct database read, configure the loader together with the existing
`[db]` settings:

```toml
[configload]
enabled=true
source="database"
namespace="my-service"
environment="production"
```

Database loading uses the local `[db]` connection and does not require the
namespace API Key because it is an in-process trusted database operation. If
the published snapshot is encrypted, the local `[configcrypt]` keyring must
also contain the key needed to decrypt it.

For HTTP loading, configure the configuration-center origin and namespace API
Key:

```toml
[configload]
enabled=true
source="http"
namespace="my-service"
environment="production"

[configload.http]
baseURL="https://config.example.com"
apiKey="replace-with-the-namespace-api-key"
timeout="5s"
```

The HTTP timeout must be greater than zero and at most one minute. Redirects
are rejected so the API Key cannot be forwarded to another origin, and the
response body is limited to 2 MiB. Prefer supplying the API Key through the
`CONFIGLOAD_HTTP_APIKEY` environment variable instead of committing it to a
configuration file.

Published values override ordinary `.app.toml` values, while Viper environment
variables retain their higher precedence. The bootstrap roots `db`,
`configcrypt`, `configcenter`, and `configload` are never accepted from the
published payload, preventing a remote configuration from changing its own
loader, database connection, keyring, or configuration-center authentication.
Redis, queue, logger, server, and other application settings are initialized
after the published configuration is merged. Loading errors stop startup rather
than silently falling back to stale local values.

For Vite hot reload, set `ui.assetMode="vite"`, run `npm run dev` in
`storage/ui`, and keep `ui.viteDevOrigin` aligned with the Vite origin. Release
builds must run the frontend build before compiling the Go binary because the
generated manifest and assets are embedded by `storage/view.go`.

Absolutely! Here’s the **updated README tutorial** including Atlas installation instructions:

---

# Database Migration with Atlas

This guide explains how to install Atlas, set up a baseline for an existing MySQL database, and manage schema migrations for multiple environments (`prod`, `dev`, `local`).

---

## 1. Install Atlas

Atlas provides prebuilt binaries for different platforms. Follow the steps below to install it:

### **macOS (Homebrew)**

```bash
brew install ariga/tap/atlas
```

### **Linux (using shell script)**

```bash
curl -sSf https://atlasgo.sh | sh
```

* This downloads and installs the latest Atlas binary in your system.

### **Verify Installation**

```bash
atlas version
```

You should see the installed Atlas version printed.

---
## 2. Configure Environment Variables (`.env`)

Atlas uses environment variables to store sensitive database credentials. Create a `.env` file in your project root:

```env
DB_DEV_USERNAME="dev"
DB_DEV_PASSWORD="devpassword"
DB_DEV_HOST="localhost"
DB_DEV_PORT="3306"
DB_DEV_NAME="devdb"

DB_LOCAL_USERNAME="localuser"
DB_LOCAL_PASSWORD="localpassword"
DB_LOCAL_HOST="localhost"
DB_LOCAL_PORT="3306"
DB_LOCAL_NAME="localdb"

DB_PROD_USERNAME="produser"
DB_PROD_PASSWORD="prodpassword"
DB_PROD_HOST="localhost"
DB_PROD_PORT="3306"
DB_PROD_NAME="productiondb"
```

### Load `.env` in your shell

For **bash/zsh**:

```bash
# Automatically export all variables in .env
set -a
source .env
set +a
```

---

## 3. Generate the Baseline

Inspect your production database and create a baseline schema snapshot:

```bash
atlas schema inspect --env prod > atlas/schema/0-baseline.hcl
```

* Generates `0-baseline.hcl`, capturing the current database structure.
* Baseline represents the starting point for Atlas migration management.

---

## 4. Generate the Initial Migration

Generate an initial migration based on the baseline:

```bash
atlas migrate diff initial --env prod
```

* Creates migration SQL in the `atlas/migrations` directory.
* SQL is generated but not yet applied.

---

## 5. Apply the Baseline

Tell Atlas that the current production database corresponds to the baseline version:

```bash
atlas migrate apply --env prod --baseline 20251120060720
```

* Replace `20251120060720` with the version number from your baseline migration file.
* Atlas records the baseline in the `atlas_schema_migrations` table.
* **No SQL will be executed**, as the database already matches the baseline.

---

## 6. Create and Apply New Migrations

After baseline setup, create new schema migrations. For example, to add auth-related tables:

```bash
atlas migrate diff create_auth_tables --env prod
```

If the migration must insert default, lookup, or other initialization data,
edit the newly generated SQL file in `atlas/migrations` before applying it.
Place data statements in dependency-safe order. For example, insert a default
row after creating its table and before backfilling existing rows or enforcing
a foreign key that references it.

After manually editing a migration file, recalculate the directory checksum,
validate the migration directory, and then apply it:

```bash
atlas migrate hash --env prod
atlas migrate validate --env prod
atlas migrate apply --env prod
```

* `migrate diff` generates the structural migration SQL.
* Data SQL such as `INSERT` statements is added manually when required.
* `migrate hash` updates `atlas/migrations/atlas.sum` after manual edits.
* `migrate validate` checks the complete migration directory before application.
* `migrate apply` executes the migration and updates the migration history.

---

## 7. Switch to Other Environments

Once the baseline is established in `prod`, you can repeat migration steps for `dev` or `local`:

1. Configure environment variables in `atlas.hcl` for each environment.
2. Generate the structural migration:

```bash
atlas migrate diff <migration_name> --env local
```

3. If initialization data is required, edit the generated SQL file and add the
   necessary data statements before applying the migration.
4. Recalculate the checksum after editing the migration file:

```bash
atlas migrate hash --env local
```

5. Validate the migration directory:

```bash
atlas migrate validate --env local
```

6. Apply the migration:

```bash
atlas migrate apply --env local
```

* Keeps all environments synchronized with the same migration history.

---

### Notes

* Always ensure the baseline file accurately reflects the current database schema.
* Baseline versions are **environment-specific**; each environment must initialize its baseline independently.
* Atlas migration only tracks schema changes; it does **not restore deleted data**. Always back up your database before applying destructive migrations.

---


**Thanks**
