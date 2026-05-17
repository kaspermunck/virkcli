---
name: virk
description: >
  Query the official Danish company registry (VIRK / CVR) for companies, persons,
  production units, and financial reports. Trigger when the user asks about Danish
  businesses, CVR numbers, P-numbers (P-enheder), company ownership, directors, board
  members, annual reports, or any data originating from VIRK / CVR / Erhvervsstyrelsen.
  Examples: "look up CVR 36945745", "who owns Lunar Bank", "what did LWOH earn last year",
  "find fintech companies in Copenhagen", "what boards does Ken Villum Klausen sit on",
  "production units under CVR 39697696", "what companies does LWOH own shares in".
tools: [Bash]
---

# virk — Danish CVR / VIRK Data Skill

You help users query the official Danish company registry (VIRK, operated by
Erhvervsstyrelsen) using the `virkcli` binary. Follow the entity-triage pipeline
below to route any question to the right subcommand, then drill deeper as needed.

## Prerequisites

- `virkcli` binary in PATH (built from `~/dev/virkcli`).
- `VIRK_USERNAME` and `VIRK_PASSWORD` must be set in the environment (HTTP Basic
  Auth against the VIRK API). The CLI errors out cleanly if they are missing — do
  not prompt the user to paste them; redirect to env vars / shell rc.

## Entity triage — pick the right command

Classify the user's input first, then dispatch:

| User has / asks about | Command |
|---|---|
| 8-digit CVR number (company ID) | `virkcli lookup <cvr>` |
| Fuzzy company name / city filter | `virkcli search <query>` |
| Person's name | `virkcli person <name>` |
| `enhedsNummer` (10-digit participant ID) | `virkcli person --id <enhedsNummer>` |
| 10-digit P-number (production unit) | `virkcli punit <pNummer>` |
| "P-units / locations for this CVR" | `virkcli punit --cvr <cvr>` |
| Annual-report figures (revenue, equity, profit) | `virkcli financials <cvr>` |
| "What companies does this CVR own / sit on the board of?" | `virkcli ejer <cvr>` |

If the input is ambiguous ("Lunar" — a company or a person?), default to
`search` first, then follow up with `lookup` on the best-matching CVR.

## Commands

### `search` — fuzzy company search

```bash
virkcli search <query> [--city <postdistrikt>] [--active] [--limit N] [--json] [--raw]
```

Matches against `Vrvirksomhed.virksomhedMetadata.nyesteNavn.navn`. Output columns:
CVR, form, status, city, name. `--active` filters to `sammensatStatus = NORMAL`
(excludes dissolved / under-liquidation companies). Use `--city "København"` or
similar (Danish postdistrikt name).

### `lookup` — full company detail

```bash
virkcli lookup <cvr> [--json] [--raw]
```

Prints company card: name, form, status, founding date, aliases (binavne),
address, municipality, industry code + text, employee range, contact info, P-unit
count, and the full deltager (participant) list with humanised roles: Reel ejer
(with ownership %), Stifter, Revisor, Ledelse, Formand, Direktør, Adm. dir.,
Bestyrelsesmedlem. Historical roles are included (previous directors show up too).

### `financials` — annual-report figures from XBRL

```bash
virkcli financials <cvr>              # latest available year
virkcli financials <cvr> --year 2024  # specific fiscal-year end calendar year
virkcli financials <cvr> --all        # full filing history (XBRL + PDF-only)
virkcli financials <cvr> --json
virkcli financials <cvr> --raw        # JSON metadata of filings found
virkcli financials <cvr> --raw-xbrl   # dump raw XBRL document
virkcli financials <cvr> --url        # print latest PDF annual report URL (combine with --year)
virkcli financials <cvr> --open       # open latest PDF annual report in default browser
```

Fields extracted: fiscalYearEnd, revenue, grossProfit, profit, equity, assets.
All amounts are DKK. Figures are extracted from XBRL filings only. **PDF-only
filings** (common for banks, IFRS reporters, and older filings) cannot be
extracted, but their URLs *are* surfaced: the default print for a PDF-only
company lists every PDF filing with its fiscal-year end and download URL, and
`--json` includes a `pdfs[]` array. `--all` marks PDF-only years with `*`.
Values are filtered to the non-dimensioned context matching the filing's
fiscal-year end, so equity/assets do not double-count per-component breakdowns.

### `person` — deltager lookups

```bash
virkcli person <name> [--limit N] [--json] [--raw]
virkcli person --id <enhedsNummer> [--active] [--json] [--raw]
```

Name search returns hits with `enhedsNummer`, relation count, and name. `--id`
fetches full detail: name, `enhedsNummer`, address (or "hidden" for privacy-
protected records), and every company relation with CVR, role, and whether the
role is currently active. `--active` (only with `--id`) filters to roles whose
`periode.gyldigTil` is unset, i.e. currently held.

Use this to build out someone's business footprint across every company they
have any registered role in (founder, owner, director, board member, chair,
auditor).

### `ejer` — reverse ownership / role lookup

```bash
virkcli ejer <cvr> [--active-only] [--limit N] [--json] [--raw]
```

Lists every company in which `<cvr>` appears as a deltager — owner, stifter,
board member, or auditor. The CVR system records each relation on the *owned*
company's record, so a holding company's portfolio (or an audit firm's client
list) is otherwise invisible. Use this any time the user asks "what does this
company own" or "what boards / clients is this firm on". Output columns: CVR,
name, role, ownership %, active. `--active-only` filters out ended relations.

### `punit` — production units (P-enheder)

```bash
virkcli punit <pNummer>         # detail for one P-unit
virkcli punit --cvr <cvr>       # list every P-unit under a CVR
virkcli punit <pNummer> --json
virkcli punit <pNummer> --raw
```

A production unit is a physical operating location. Each is tied to exactly one
CVR (parent company) and has its own industry code, address, employee range,
and contact info, which can differ from the parent.

## Common workflows

**1. Who's behind this company?**
```bash
virkcli lookup <cvr>    # deltagere are in the bottom of the card
```

**2. Trace a person across every company they're involved with.**
```bash
virkcli person "<name>"                # find their enhedsNummer
virkcli person --id <enhedsNr> --active  # list all current roles
```

**3. Find a company by name, then drill in.**
```bash
virkcli search "<name>" --active    # get a CVR shortlist
virkcli lookup <cvr>                # full company detail
virkcli financials <cvr> --all      # historical P&L / balance sheet
virkcli punit --cvr <cvr>           # physical locations
```

**4. What is this P-number and what company does it belong to?**
```bash
virkcli punit <pNummer>   # shows parent CVR
virkcli lookup <cvr>      # drill into the parent
```

**5. Historical financial trend for a company.**
```bash
virkcli financials <cvr> --all
```

**6. Map a holding company's portfolio (reverse ownership).**
```bash
virkcli ejer <cvr> --active-only   # current portfolio + board seats + audits
```

## Global flags (present on every command)

- `--json` — parsed result as JSON (scriptable, pipe to `jq`).
- `--raw` — raw Elasticsearch response body (for debugging or custom parsing).

Never combine `--json` and `--raw` — pick one. Most answers should use the
plain table output; only use `--json` when the user asks for structured data
or you need to pipe into another tool.

## Presenting results

- For a single entity (company, person, P-unit): relay the formatted output
  more or less verbatim, but lead with a one-line summary answering the user's
  question.
- For lists (search hits, `financials --all`, P-unit lists): render the key
  columns as a markdown table in your reply; note the row count.
- For financials: always state the currency (DKK) and the fiscal-year end.
- Cite the data source as Erhvervsstyrelsen / VIRK (data.virk.dk).

## Error recovery

**"VIRK_USERNAME and VIRK_PASSWORD must be set"** — tell the user to export
these in their shell env; do not ask them to paste credentials into the chat.

**"company not found" / "no production unit found"** — verify the CVR /
P-number is valid (8 digits for CVR, 10 digits for P-number). Consider whether
the user gave a name instead and route to `search` / `person`.

**"no person found"** on `--id` — enhedsNummer may be wrong; run a name search
first to confirm.

**`financials --all` shows all dashes for some years** — those years are
PDF-only filings (marked with `*`). Figures cannot be extracted from PDFs.
This is common for banks, IFRS reporters, and older filings. Note this to the
user and mention which years do have XBRL data. Offer the PDF URL via
`virkcli financials <cvr> --url` (most recent) or `--url --year <YYYY>` for a
specific year, or `--open` to launch it in the browser.

**`--year` returns nothing** — the year must be the fiscal-year *end* calendar
year, not the accounting period start. E.g. fiscal year 2024-07-01 to
2025-06-30 is `--year 2025`.

**HTTP 400 from VIRK** — usually a transient API issue or a malformed query.
Rerun; if it persists, try `--raw` to see the upstream response.

## Example walkthrough

**User:** "Who sits on the board of Lunar Bank, and what did they earn last year?"

```bash
# Step 1: No CVR given → search
virkcli search "Lunar Bank" --active
# → CVR 39697696, Lunar Bank A/S, Aarhus C

# Step 2: Lookup for deltagere
virkcli lookup 39697696
# → deltagere section lists board members (Bestyrelsesmedlem) and
#   the chair (Formand), plus the current Adm. dir.

# Step 3: Financials
virkcli financials 39697696 --all
# → table of every XBRL year with revenue / profit / equity / assets in DKK
```

Respond with: the current board (filtered to `Bestyrelsesmedlem` + `Formand`
rows from the `lookup` output), and a compact financial-history table pulling
the most recent year to the top. Cite VIRK as the source.
