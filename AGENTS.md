# Basic requirements for Golang programs.

You are an expert AI programming assistant specializing in building APIs with Go, using the Gin library to build http api for services.

Always use the latest stable version of Go (1.23 or 1.24 or newer) and be familiar with RESTful API design principles, best practices, and Go idioms.

- You need to use the module name defined in go.mod as the base package name for this project.
- The programs you write should follow the style of the existing code to maintain overall consistency and simplicity in the codebase. You should adhere to Golang best practices when writing your programs.
- First think step-by-step - describe your plan for the API structure, endpoints, and data flow in pseudocode, written out in great detail.
- Confirm the plan, then write code!
- Write correct, up-to-date, bug-free, fully functional, secure, and efficient Go code for APIs.
- Utilize the new Gin framework for routing
- Implement proper handling of different HTTP methods (GET, POST, PUT, DELETE, etc.)
- Implement proper error handling, including custom error types when beneficial.
- Do not return errors with `fmt.Errorf`. Define package-level sentinel errors with `errors.New`, return them directly when no underlying cause exists, and use `errors.Join(ErrSpecific, err)` when preserving an underlying error. This keeps errors testable and classifiable with `errors.Is` / `errors.As`.
- Use appropriate status codes and format JSON responses correctly.
- Implement input validation for API endpoints.
- Utilize Go's built-in concurrency features when beneficial for API performance.
- Follow RESTful API design principles and best practices.
- Include necessary imports, package declarations, and any required setup code.
- Implement proper logging using the `github.com/rs/zerolog/log` package.
- Consider implementing middleware for cross-cutting concerns (e.g., logging, authentication).
- Implement rate limiting and authentication/authorization when appropriate, using standard library features or simple custom implementations.
- Leave NO todos, placeholders, or missing pieces in the API implementation.
- Be concise in explanations, but provide brief comments for complex logic or Go-specific idioms.
- If unsure about a best practice or implementation detail, say so instead of guessing.
- Offer suggestions for testing the API endpoints using Go's testing package.
- The programs you write must follow cybersecurity standards and protect user privacy.
- Avoid using hardcoded values in the code; instead, use defined static variables or configuration files for settings whenever possible.
- Avoid duplicated implementation logic. When two or more code paths share the same lifecycle, validation, query preparation, file output, cleanup, cache, decoding, or result-building logic, extract a small helper function or focused type instead of copying the block. Keep the abstraction narrow and type-safe, pass package-level sentinel errors into helpers when each caller needs its own error classification, and verify duplicate-code changes with `golangci-lint run --enable-only=dupl` in addition to the full lint run.
- Use each tool's system default cache directories. Do not set `GOCACHE`, `GOMODCACHE`, `GOLANGCI_LINT_CACHE`, npm cache paths, or similar cache environment variables to paths inside this repository unless the user explicitly asks for an isolated per-project cache. Shared default caches allow Go, golangci-lint, npm, and other tools to reuse artifacts across projects and avoid duplicating large cache directories in every checkout.
- For more requirements, please refer to the file and directory descriptions as well as the architecture description of this project.

Always prioritize security, scalability, and maintainability in your API designs and implementations. Leverage the power and simplicity of Go's standard library to create efficient and idiomatic APIs.

# Architecture description of this project

This project supports Go modules, and the module information is defined in the `/go.mod` file.

This project is a Golang-based project that uses `github.com/redis/go-redis/v9` as the queue and caching system, and employs the `github.com/go-redis/cache/v9` library for simple cache management and operations. The routing system of this project is based on the `Gin` framework, and the ORM system uses `GORM` with `MySQL` as the database. The queue task framework in this project utilizes `github.com/ThreeDotsLabs/watermill`. The logging system is based on the `github.com/rs/zerolog/log` framework. The command-line program framework uses `github.com/spf13/cobra`. The programs written for this project must pass the `golangci-lint` checks; please refer to the `/.golangci.yaml` file for the requirements. If this project needs to call third-party HTTP interfaces, please use the `github.com/go-resty/resty/v2` library, which provides a well-structured wrapper for HTTP calls.This project uses `github.com/spf13/viper` to read configuration files and environment variables.

You need to organize the programs you write according to the file structure of this project, maintaining a clean and consistent directory structure. Moreover, you should prioritize using the methods defined in to return HTTP responses and data with the correct status codes.

Please refer to the following handler example when writing HTTP handlers.
```golang
func HomePage(c *base.Context) {
    // access the user if user logined
	user := c.LoginUser()
	if user == nil {
        // if not logined, return 403 with forbidden status code
		c.Forbidden("you are not login", gin.H{})
	} else {
        // if logined, return 200 and the user infomation
		c.Success(gin.H{
			"data": fmt.Sprintf("your user id is: %d", user.ID),
		})
	}
}
```
`base.Context` defines commonly used HTTP response methods, including both normal and error responses, as well as functions for globally accessing user information.

After defining the handler, you also need to add it to the mapping in `/api/route/route.go`.
```golang
r.GET("/", handler.HomePage)
```

# Directory or files description for this project

- `/go.mod` : The Go Mod file defines the package name of this Go project as well as its third-party dependencies. You need to use this package name in your code to correctly reference the Go packages defined in the project, and declare and record any third-party packages you use according to the Go Mod specification in this file.
- `/main.go` : The entry file of this project, it is generated by cobra, should not be changed.
- `/api/base/context.go` : The custom context extends gin.Context, I added useful function to the context. You need to first analyze the program in `/api/base/context.go` and prioritize using the various methods defined in `/api/base/context.go` to handle normal and error responses.
- `/api/route/route.go` : Route definition file, all routes are added to this file and can be defined by different functions or categories inside different function groups. Please use `*base.Router` as the route object. You should only write route mapping-related code in `/api/route/route.go` and must not write object initialization or feature initialization code there.
- `/api/route/middleware.go` : Based on the gin framework's middleware, all new middleware should be placed in this file. And use `base.Context` as the context for requests in the middleware.
- `api/server/server.go` : Define and start the gin.Server program, generally do not need to be modified, usually only modify which involves the `github.com/gin-contrib/cors` library to define the cross-domain related configuration, such as adding the domain name, request headers, exposed response headers, or dynamically determine whether the domain name is allowed to cross-domain through the AllowOriginFunc
- `/queue/job/job.go` : This file defines the Job interface for queue jobs. When using queue tasks, you need to implement each method of this interface, define the job name, specify the queue to use, encode and decode the job data, and define the job's execution method. After a Job is fully defined, it must be registered in the `/queue/subscriber.go` file.
- /.app.toml : This file is used to store configuration information in TOML format. It can be used to store common configuration settings or environment variables. `/.app.toml.example` is a sample template for the configuration file. When adding or modifying configurations, you need to provide examples in the template.
- /api/handler : This directory is used to store all handler programs that process HTTP requests. All handlers need to use base.Context as a parameter and be properly mapped in the `/api/route/route.go` file.
- /cache : This directory is used to store all programs related to cache operations. The cache uses Redis for data storage. This directory encapsulates all Redis operations through the use of the cache. The cache is implemented using the `github.com/go-redis/cache/v9` library.
- /cmd : This directory is used to store command-line programs, utilizing the `github.com/spf13/cobra` library. It allows invoking non-HTTP functionalities from the command line.
- /db : This directory is db package, used to store programs related to the database and ORM. All ORM-related types and operations should be placed in this directory. We use the `gorm.io/gorm` library as the underlying ORM, and database access via GORM can be obtained through the `db` variable. The `db` variable is globally accessible within the `db` package. In other packages, you can obtain a reference to `db` by calling `DB()` to use the ORM. You should try to avoid performing direct `db` operations outside of the `db` package.
- /queue/job : This directory is used to store all programs related to queue jobs. For the Job interface specification, please refer to the description in `/queue/job/job.go` .
- /redis : This directory is redis package, used to store all wrapper programs that directly operate on Redis data. It is important to note that when operating on Redis keys, you should use the `WithPrefix` function to add a prefix to the key, and this prefix is defined through the configuration file. The GetClient method in the redis package can be used to obtain a redis.Client object for operating on Redis. You can use GetClient in external packages. However, you should try to write Redis operation programs within the redis package for reuse in other packages. If it is a common caching operation, you should prioritize using the cache package.
  Feature packages must not directly import or call `github.com/redis/go-redis/v9`; add wrapper functions in `/redis` and call those wrappers instead. Redis key prefixing must happen inside the redis package wrappers whenever possible.
- /services : This directory is used to store programs that call functionalities of third-party or external systems. Programs that can exist relatively independently can also be placed in this folder.
- /xutil : This folder is used to store custom utility functions. Independent and reusable utility functions can be placed here. This folder already includes encryption and decryption functions as well as some common string processing functions.
- /storage/static : This directory is used to store static asset files, which can include HTML, JavaScript, CSS, images, etc. These files will be bundled into the Golang binary during the compilation process.
- /storage/views : This directory is used to store HTML files that will be rendered by Golang. HTML files can be categorized into subfolders based on their functionality. Place basic layout files in the `/storage/views/layout` directory, shared components in the `/storage/views/shared` directory, and other specific pages can be categorized according to their functional modules. To automatically reference the layout, page files need to follow the naming convention. The convention is as follows: If the base layout file is named `mylayout.html`, the functional page file should be named `feature.mylayout.html`, i.e., `featureName` + `layoutFileName`.
- /storage/ui : This directory is used to store programs related to front-end frameworks, such as independent programs for Vue, React, and other front-end projects.

# Frontend Architecture and Backend Deployment

The current browser UI is a Vite-based multi-entry application under `/storage/ui`, rendered by Go HTML templates under `/storage/views`, and bundled into the Go binary through `/storage/view.go`.

- Frontend source code lives under `/storage/ui/src/<feature>/`. Each page owns its `main.js` and usually a colocated `main.css`. Existing entries are `src/recorder/main.js`, `src/codec-lab/main.js`, and `src/audio-archives/main.js`.
- The frontend uses Vite, Tailwind CSS v4 via `@tailwindcss/vite`, and Alpine.js. Keep this lightweight page-level architecture unless a feature clearly needs a larger framework. Do not introduce React, Vue, or another SPA framework for ordinary pages.
- Use Alpine.js as the page interaction and state-management mechanism. Define page state and actions in a page-local `x-data` factory, initialize through `x-init` or Alpine `init()`, bind form state with `x-model`, render state with `x-show`/`x-if`/`x-for`, and use Alpine event directives such as `@click`, `@submit.prevent`, `@change`, `@keydown.escape.window`, and their modifiers instead of adding manual DOM listeners for ordinary page interactions. Use `$watch`, `$nextTick`, `$refs`, and `$dispatch` for reactive coordination. When an external browser API or an imperative third-party widget requires `addEventListener`, register it during Alpine initialization and remove it in Alpine `destroy()`; prefer `@event.window`/`@event.document` when Alpine can manage the listener lifecycle.
- Alpine reactive state is implemented with JavaScript `Proxy` objects. Never pass Alpine state, nested reactive values, `$data`, or objects read from Alpine getters directly to `structuredClone`, `postMessage`, History state, or another API that rejects proxies; this causes `DataCloneError`. Convert supported configuration values to plain objects and arrays first with a focused recursive helper such as `cloneConfigValue`, or unwrap and normalize the complete nested value explicitly. Keep derived Alpine getters pure, do not mutate reactive state while rendering, and clone plain configuration data before an imperative editor mutates it. Store DOM nodes in `$refs`, and keep media streams, sockets, timers, class instances, and other non-reactive resources outside Alpine state when possible; if they must be referenced by Alpine, explicitly prevent accidental proxy-dependent cloning and clean them up in `destroy()`.
- Tailwind CSS v4 does not automatically discover the Go HTML templates under `/storage/views` when the stylesheet is compiled from `/storage/ui`. The `includeGoTemplateSources` plugin in `storage/ui/vite.config.js` globally injects the `/storage/views` source directory into every stylesheet that imports Tailwind. Do not add page-local `@source` directives or replace the registered directory with an HTML glob. Tailwind v4 recursively scans the registered directory, while a glob can be treated as a non-matching source path. Keep the global Vite plugin and the post-build CSS verification enabled; otherwise component CSS may load while layout utilities such as `grid`, `flex`, spacing, widths, and responsive variants are silently omitted.
- Configuration-center diffs use app-specific Shiki language and Pierre theme bundles through exact aliases in `storage/ui/vite.config.js`. Keep syntax highlighting limited to `dotenv`, `ini`, `json`, `yaml`, and `toml`, and keep only the `pierre-light` theme unless the UI explicitly adds another supported format or theme. Do not import the top-level full Shiki bundle directly; it emits every grammar and theme as build chunks. Keep `verify-config-center-bundle.mjs` in the production build so dependency upgrades cannot silently restore the full bundle.
- After adding or changing Tailwind classes in a Go template, run `npm run build` inside `/storage/ui` and verify more than build success: confirm representative template utilities exist in the generated CSS or inspect the page's computed layout in a browser (for example, expected Grid/Flex display and padding). A successful Vite build alone does not prove that external templates were scanned.
- `storage/ui/vite.config.js` is the source of truth for frontend entry points. When adding a new page, add its entry to `build.rollupOptions.input`; the manifest key must match the string used by the Go handler, such as `src/feature/main.js`.
- Go page handlers should define a package-level entry constant, call `c.GetCSSJsFiles(entry)`, and pass `title`, `use_vite`, `vite_origin`, `css_files`, and `js_files` into the template. Follow the existing page-handler pattern.
- HTML templates select assets with `use_vite`: when true, include `{{ .vite_origin }}/@vite/client` and the Vite source entry; otherwise include the CSS and JS files resolved from `storage/ui/dist/manifest.json` and served by the Go backend.
- Development defaults to embedded Vite build assets when a manifest is available, so a single Go server can render a fully styled page. For Vite hot reload, set `ui.assetMode="vite"`, run `npm run dev` inside `/storage/ui`, and configure `ui.viteDevOrigin` when it differs from `http://127.0.0.1:5173`.
- Release deployment requires building the frontend before building or packaging the Go binary: run `npm run build` inside `/storage/ui`, then build the Go program. `/storage/view.go` embeds `views`, `ui/dist`, and `static`, so missing `storage/ui/dist/manifest.json` means release templates cannot resolve hashed assets.
- Generated Vite output goes to `/storage/ui/dist` with hashed assets under `/storage/ui/dist/assets`. Keep `dist/keep.txt` as the placeholder that preserves the embedded directory, but do not commit generated `manifest.json` or hashed assets unless a release process explicitly requires checked-in build artifacts.
- Browser assets from Vite are served by the backend at `/assets/*filepath` from the embedded `storage.Assets` filesystem. Use relative URLs or backend-origin URLs for API calls and assets so the same page works in development, release, and reverse-proxy deployments.
- Root static files in `/storage/static` are served by `ServeRootStaticFiles` as fallback files, not as Vite build output. Put long-lived manually maintained root files there only when they should be directly addressable from `/filename`.
- Route definitions stay in `/api/route/route.go`; frontend page handlers and their JSON or WebSocket APIs stay in `/api/handler`. Do not put initialization logic in the route file.
- Frontend API calls should use the existing backend response shape (`status`, `message`, `data`) and relative paths such as `/audio-archives/api/sessions`. Use `location.host` and the current scheme when constructing WebSocket URLs.
- Keep Alpine app factories page-local and expose only the template entry function on `window`, matching `window.recorderApp`, `window.codecLabApp`, and `window.audioArchiveApp`. Revoke object URLs and clean up media streams, worklets, WebSocket connections, and audio contexts when a page flow stops or resets.
- Use `package-lock.json` for deterministic npm installs. Do not commit `node_modules`, tool caches, or per-machine build caches, and do not redirect npm, Vite, Go, or golangci-lint caches into this repository.
- Configuration-center drafts, descriptions, published payloads, and namespace API Keys use the `services/configcrypt` envelope-encryption service. Never store raw encryption keys or plaintext namespace API Keys in `.app.toml`, environment namespaces, the database, source code, logs, or Git. Application configuration may contain only the provider, active key ID, and an absolute external keyring path. Preserve ciphertext versioning and AAD context, retain old key IDs for reads during rotation, and use `config-key reencrypt` before retiring a key.
- The configuration center is disabled unless `configcenter.enabled=true`. When enabled, startup validation requires the configured Basic Auth username and password. Apply `gin.BasicAuth()` to the `/config-center` management page and management APIs; do not implement custom Basic Auth credential parsing or comparison. Do not apply Basic Auth to `/config-center/api/runtime/*`; runtime endpoints use the namespace `X-API-Key` and still require the global configuration-center switch. Disabled configuration-center routes must return `404`, and Basic Auth must be deployed only behind HTTPS.
- Published configuration may be merged into Viper from the root command before child command hooks run through `[configload]`, using either the trusted direct database source or the runtime HTTP API. Keep Cobra parent hook traversal enabled so every child command receives the merged configuration. Initialize `configcrypt` once in that same root hook rather than repeating it in child commands. Only `config-key generate` with the database source may skip remote loading because it bootstraps the first keyring, while the HTTP source remains available. Load only immutable published snapshots, fail command startup on load errors, preserve Viper environment-variable precedence, and never allow a published payload to override the bootstrap roots `db`, `configcrypt`, `configcenter`, or `configload`. The database source uses local database and keyring settings; the HTTP source uses the namespace `X-API-Key`, rejects redirects, and enforces timeout and response-size limits.

# Database Table Definitions and Database Migrations
1. When you need to modify database tables or fields, use the atlas tool to define the tables and generate migration files. Database tables should be defined in the `/atlas/schema` directory, and it is recommended to define tables for related modules in a single file. The configuration file for the atlas tool is located in this repository at `/atlas.hcl`.
2. In this project, the database tables prefix is defined in the `.app.toml`, should use the prefix when define tables.
3. GORM database table definitions must follow these conventions:
   - Use plural table names with the table prefix.
   - Every table must have a single `id` primary key using auto-increment semantics. Do not use business keys or compound keys as the primary key.
   - Every table must include `created_at` and `updated_at` fields.
   - Do not define nullable columns. Use explicit defaults instead: empty string for strings, `0` for numbers, `false` or a clear default for booleans, and explicit status defaults when applicable.
   - Business uniqueness must be enforced with unique indexes instead of primary keys.
   - Avoid redundant indexes. If a left prefix of a composite index already covers the query pattern, do not add a separate duplicate prefix index.
   - Keep Atlas schema, generated migrations, and GORM model structs consistent with each other.
4. Generate migrations:

```bash
atlas migrate diff <migration_name> --env local
```

5. If the migration needs default, lookup, or other initialization data, edit the newly generated migration SQL file before applying it. Add the required `INSERT` or other data-migration statements in dependency-safe order; for example, insert a referenced default row after its table is created and before existing rows are backfilled or a foreign key is enforced. Do not expect `atlas migrate diff` to infer ordinary seed data from table definitions.
6. After any manual edit to a generated migration file, recalculate the migration directory checksum:

```bash
atlas migrate hash --env local
```

7. Validate the complete migration directory before applying it:

```bash
atlas migrate validate --env local
```

8. Apply migrations only after the generated SQL, manually added data SQL, checksum, and validation have all been reviewed:

```bash
atlas migrate apply --env local
```

# CRUD actions for GORM model
When you need to provide CRUD operations for a database model (GORM) struct, you can quickly implement them as follows:
1. Have the model implement the `IDModel` interface in `db/crud_core.go`, providing the model’s ID and the loading method for related GORM models.
2. When creating or updating data, create an input form structure that implements the `InputConverter` interface in `db/crud_core.go` to convert the input data into GORM model data.
3. When reading Gorm model data for display, implement the `ViewConverter` interface in `db/crud_core.go` for the output data structure to convert the Gorm model into the output structure.
4. When providing search or listing services, you need to support search and filtering capabilities. In this case, have the input form data structure implement the `Searcher` interface in `db/crud_core.go` to dynamically add search and filter conditions.
5. Finally, provide a CRUD service corresponding to the Model, register it using `NewCrudController`, and add it to the routing.

Below is an implementation of CRUD operations for a user model:
```go
type User struct {
    gorm.Model
    AccountID uint `gorm:"index" json:"account_id"`
    Account   *Account
}

var _ IDModel = &User{}

func (u *User) GetID() uint { return u.ID }

func (u *User) LoadRelation(ctx context.Context, relations ...string) {
    for _, relation := range relations {
        db.WithContext(ctx).Model(u).Association(relation).Find(u.Account)
    }
}
```
The CRUD service and controller for User Model:
```go

type UserCreateForm struct {
	AccountID uint `form:"account_id" json:"account_id" binding:"required"`
}

func (u *UserCreateForm) ToModel() *db.User {
	return &db.User{
		AccountID: u.AccountID,
	}
}

type UserUpdateForm struct {
	Name      string `form:"name" json:"name" binding:"required"`
	AccountID uint   `form:"account_id" json:"account_id" binding:"required"`
}

func (u *UserUpdateForm) ToModel() *db.User {
	p := &db.User{
		AccountID: u.AccountID,
	}
	p.ID = 0
	return p
}

type UserPublicView struct {
	Name      string `json:"name"`
	AccountID uint   `json:"account_id"`
	ID        uint   `json:"id"`
}

func (u *UserPublicView) FromModel(p *db.User) *UserPublicView {
	return &UserPublicView{
		ID:        p.ID,
		Name:      "",
		AccountID: p.AccountID,
	}
}

func (u *UserPublicView) Preloads() []string {
	return []string{}
}

type UserSearchInput struct {
	AccountID uint `form:"account_id"`
}

func (u *UserSearchInput) GetScopes() []func(*gorm.DB) *gorm.DB {
	var scopes []func(*gorm.DB) *gorm.DB

	// 1. Also, You can add preloads to *gorm.DB at here.

	// 2. Filter by account_id
	if u.AccountID > 0 {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("account_id", u.AccountID)
		})
	}

	return scopes
}

func UserControl(r *base.Router) {
	// Generic parameters：Model, Create, Update, View, Search
	providerService := db.NewCrudService[
		*db.User,
		*UserCreateForm,
		*UserUpdateForm,
		*UserPublicView,
		*UserSearchInput,
	](
		func() *db.User {
			return &db.User{}
		},
		func() *UserPublicView {
			return &UserPublicView{}
		},
		func() *UserSearchInput {
			return &UserSearchInput{}
		},
	)
	providerCtrl := NewCrudController(providerService)
	providerCtrl.RegisterRoutes(r)
}
```
Related files for CRUD services are: `/api/handler/crud_base.go`, `/db/crud_core.go`, `/db/crud_pagination.go`, `/db/crud_service.go`
