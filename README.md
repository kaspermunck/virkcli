# virkcli

CLI for the official Danish [VIRK](https://datacvr.virk.dk) / CVR registry — Erhvervsstyrelsen's public company database.

## Coverage

| Subcommand | What it does |
|---|---|
| `virkcli lookup <cvr>` | Company detail: name, status, founded, address, industry, contacts, owners, board, auditors |
| `virkcli search <query>` | Search companies by name (active only by default) |
| `virkcli financials <cvr>` | Annual report figures (revenue, profit, equity, assets) extracted from XBRL filings |
| `virkcli person <name>` | Search persons by name; look up roles by `enhedsNummer` |
| `virkcli punit <p-number>` | Production unit lookup; list P-units for a CVR |

Financial data is extracted from XBRL filings. PDF-only annual reports (common for banks, IFRS reporters, older filings) are listed but figures are not extracted — they appear as rows with a `*` marker and empty value columns.

## Install

Via Homebrew (recommended):

```sh
brew install kaspermunck/tap/virkcli
```

From source:

```sh
go install github.com/kaspermunck/virkcli@latest
```

## Claude Code skill

A [Claude Code](https://claude.com/claude-code) skill ships inside the Homebrew formula. After `brew install`, enable it once with the symlink Homebrew prints in its caveats:

```sh
mkdir -p ~/.claude/skills && ln -sfn "$(brew --prefix virkcli)/share/virkcli/skill" ~/.claude/skills/virk
```

Re-run `brew upgrade virkcli` to update both the binary and the skill atomically.

## Auth

VIRK's data distribution endpoint requires Basic Auth credentials. Sign up at [data.virk.dk](https://data.virk.dk) → "Distribution" to receive a username + password.

Set them in your environment:

```sh
export VIRK_USERNAME="<your-username>"
export VIRK_PASSWORD="<your-password>"
```

macOS Keychain pattern (recommended — keeps secrets off disk):

```sh
security add-generic-password -s "virkcli" -a "VIRK_USERNAME" -w '<your-username>' -U
security add-generic-password -s "virkcli" -a "VIRK_PASSWORD" -w '<your-password>' -U
```

`virkcli` reads the env vars first; if either is empty, it falls back to the same keychain entries directly, so no shell-rc export is needed.

## Usage

```sh
# Look up a company by CVR
virkcli lookup 24256790

# Search by name
virkcli search "Novo Nordisk"

# Multi-year financial history (XBRL only)
virkcli financials 24256790 --all

# Find a person by name
virkcli person "Mette Frederiksen"

# Detailed person record by enhedsNummer
virkcli person --id 4000123456

# Production units for a CVR
virkcli punit --cvr 24256790

# JSON output for downstream tooling
virkcli lookup 24256790 --json

# Wrap output in the shared envelope ({source, kind, version, data, fetchedAt})
virkcli lookup 24256790 --envelope
```

## Output formats

Every subcommand supports three output modes:

- **(default)** — human-readable table / tree.
- `--json` — parsed, flattened JSON suitable for piping into `jq`.
- `--envelope` — JSON wrapped in `{source, kind, version, data, fetchedAt}` for downstream tools that consume multiple data sources uniformly.

## Notes

- VIRK redacts the Revenue field for consolidated group accounts (koncernregnskaber) for many large A/S filings. When Revenue is missing but other fields populated, that's an upstream data quirk, not a `virkcli` bug.
- Person names in Denmark are not unique. When `virkcli person <name>` returns multiple matches, always confirm the `enhedsNummer` before treating it as authoritative — CVRs are unique, names aren't.

## Acknowledgements

`virkcli` accesses data from [Det Centrale Virksomhedsregister (CVR)](https://datacvr.virk.dk) operated by [Erhvervsstyrelsen](https://erhvervsstyrelsen.dk) (Danish Business Authority). The distribution endpoint at `distribution.virk.dk` is documented at [data.virk.dk](https://data.virk.dk).

`virkcli` is an independent open-source tool and is not affiliated with or endorsed by Erhvervsstyrelsen.

When publishing work derived from CVR data, credit the source:

> Kilde: CVR / Erhvervsstyrelsen

### Personal data note

CVR exposes personal data about company officers, beneficial owners (reelle ejere), and founders. This is by design — CVR is a public registry — but the data is still personal data under the GDPR. If you process it programmatically, observe the principles of necessity and proportionality, and consult Erhvervsstyrelsen's [distribution terms](https://data.virk.dk) before bulk or commercial reuse.

## License

This software is released under the [MIT License](LICENSE). CVR data is published by Erhvervsstyrelsen under separate terms (see [data.virk.dk](https://data.virk.dk)).
