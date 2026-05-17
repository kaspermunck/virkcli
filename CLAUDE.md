# virkcli

A CLI for querying Danish company data via the official VIRK API.

## Stack

- Go, structured as a cobra CLI
- `virk/` package: VIRK API clients (company lookup + XBRL financial parsing)
- `cmd/` package: cobra subcommands
- Auth via env vars `VIRK_USERNAME` + `VIRK_PASSWORD`, with macOS Keychain fallback (service `virkcli` / accounts `VIRK_USERNAME` and `VIRK_PASSWORD`)

## CLI commands

```
virkcli lookup <cvr>        # company detail (deltagere, address, industry, contacts)
virkcli search <query>      # fuzzy company search (--city, --active, --limit)
virkcli financials <cvr>    # annual report figures from XBRL (--year, --all); PDF-only filings listed but not extracted
virkcli person <name>       # deltager search; --id <enhedsNummer> for detail + --active
virkcli punit <pNr>         # production unit detail; --cvr to list a company's P-units
virkcli ejer <cvr>          # reverse ownership: companies where <cvr> is registered as deltager (--active-only)
```

Every command supports `--raw` (raw Elasticsearch body) and `--json` (parsed struct).

Build: `go build -o virkcli .`
Install to PATH: `go install .` (binary lands in `~/go/bin`).

## Skill

A single fat Claude Code skill `virk` ships in this repo at
`.claude/skills/virk/SKILL.md` and wraps all subcommands — Claude triages the
user's question to the right one. See the README for install instructions.

## VIRK API endpoints

- Company lookup/search: `POST http://distribution.virk.dk/cvr-permanent/virksomhed/_search`
- Persons (deltagere):   `POST http://distribution.virk.dk/cvr-permanent/deltager/_search`
- Production units:      `POST http://distribution.virk.dk/cvr-permanent/produktionsenhed/_search`
- Annual reports:        `POST http://distribution.virk.dk/offentliggoerelser/_search`
- Auth: HTTP Basic Auth

## Memory

**Always read `.claude/MEMORY.md` at the start of every session.**
Write new memories to `.claude/memory/` and update the index in `.claude/MEMORY.md`.
