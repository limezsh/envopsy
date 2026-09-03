# Envopsy

Envopsy checks whether a project's **documented environment variables match what the code actually reads**.

It is a local, single-binary CLI. It does not lint `.env` syntax, scan for secret *values*, or talk to a network.

```
envopsy 0.1.0  3 files  4 vars

error  missing-example  DENO_REGION
       src/index.js:4

error  missing-example  VITE_API_URL
       src/index.js:3

warning  unused-example  UNUSED_WEBHOOK
       .env.example:3

2 errors, 1 warning
```

## Install

```bash
go install github.com/limezsh/envopsy/cmd/envopsy@latest
```

Release binaries (linux / macOS / Windows, amd64 and arm64) are attached to GitHub Releases.

```bash
envopsy              # scan the current directory
envopsy ./path       # scan a project root
envopsy --json       # CI / tooling
envopsy --fix        # append missing keys as empty values to .env.example
```

## What it reports

| Rule | Severity | Meaning |
|---|---|---|
| `missing-example` | error | Used in code, absent from `.env.example` / `.env.sample` / `.env.template` |
| `unused-example` | warning | Documented, never referenced in code |
| `no-example-file` | warning | Code uses env vars but no template file exists |
| `tracked-env` | warning | A local `.env` file is tracked by git |
| `dynamic-access` | info | `process.env[key]` / `os.getenv(f"...")` — name cannot be resolved |
| `undocumented-local` | warning | Key in `.env` but not in the example file (`--check-local` only) |

Default `--fail-on warning` makes CI fail on errors and warnings. Infos never fail. `--fail-on never` is a dry run.

## Detection

Conservative, extension-dispatched patterns. Envopsy does **not** scan shell, Makefiles, YAML, Dockerfiles, or GitHub Actions — those sources are a false-positive factory.

| Language | Files | Matches |
|---|---|---|
| JS / TS | `.js .jsx .mjs .cjs .ts .tsx` | `process.env.NAME`, `process.env['NAME']`, `import.meta.env.NAME`, `Deno.env.get('NAME')` |
| Python | `.py` | `os.getenv`, `os.environ['NAME']`, `os.environ.get` |
| Go | `.go` | `os.Getenv`, `os.LookupEnv` |
| Ruby | `.rb` | `ENV['NAME']`, `ENV.fetch` |
| PHP | `.php` | `getenv`, `$_ENV`, Laravel `env()` — not `$_SERVER` |
| Rust | `.rs` | `env::var`, `option_env!` |
| Prisma | `.prisma` | `env("NAME")` |

Platform variables such as `PATH`, `HOME`, `NODE_ENV`, and Vite's `MODE` / `DEV` / `PROD` are ignored by default.

`.gitignore` is respected (plus `node_modules`, `vendor`, `dist`, …). Dotenv files are still read when gitignored so `--check-local` works.

## Flags

```
--json              JSON report
--fix               append missing keys as KEY= (never copies values)
--fail-on LEVEL     error | warning | never   (default: warning)
--ignore NAME       extra variable to ignore (repeatable)
--config PATH       .envopsy.toml (otherwise walk-up from the scan root)
--check-local       flag keys in .env that are not in the example file
--no-ignore         do not apply .gitignore (default skip dirs still apply)
-q, --quiet         exit code only
-v, --version
```

Exit codes: `0` clean, `1` findings at/above `--fail-on`, `2` tool error.

## Config

Optional `.envopsy.toml` in the project (or a parent directory):

```toml
fail_on = "warning"
ignore_vars = ["MY_FRAMEWORK_VAR"]
ignore_paths = ["generated/"]
example_files = [".env.example"]
```

## Security

- Fully local. No network, no telemetry.
- Dotenv **values are discarded at parse time** and never printed or written to JSON.
- `--fix` only appends empty `KEY=` lines. It will not copy secrets from `.env`.
- Project code is never executed. Symlinks that point outside the scan root are not followed.
- `tracked-env` uses `git ls-files` when git is available; missing git is not an error.

## Compared to dotenv-linter

[dotenv-linter](https://github.com/dotenv-linter/dotenv-linter) checks the *syntax and style* of `.env` files (duplicates, unordered keys, quotes). Envopsy does not. It compares **code usage** to the documented example file. Use both if you want both jobs done.

## License

MIT
