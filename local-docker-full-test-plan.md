# Local Docker Full Test Plan

Last updated: 2026-06-20 04:10 Asia/Shanghai

## Goal

Rebuild the local Docker validation flow so it is repeatable, single-candidate,
residue-audited, and strong enough to block bad changes before any GitHub
release or FnOS deployment.

## Hard Rules

1. Use exactly one local candidate image tag for active validation:
   - `chinesesubfinder:local-candidate`
2. Use exactly one isolated local test root:
   - `D:\tmp\csf-local-candidate`
3. Do not use FnOS as a debug loop.
4. Every test round must produce:
   - a pre-run residue report
   - a post-run residue report
   - an explicit keep/delete list
5. No new ad hoc image tags like `delivery-v2`, `delivery-v3`, `currentfix`.
6. No new test outputs outside `D:\tmp\csf-local-candidate` unless a command
   itself is read-only.
7. Residue cleanup suggestions must only target `csf*` local artifacts for this
   project by default; unrelated `D:\tmp` project outputs are out of scope
   unless explicitly approved.

## Allowed Local Runtime Surfaces

- Local Docker image build from current repo
- Local Docker container from `chinesesubfinder:local-candidate`
- Local bind-mounted media/test fixtures under `D:\tmp\csf-local-candidate`
- Local frontend/browser verification against the same candidate container

## Required Validation Scope

### 1. Static / Unit Gate

- `go test ./pkg/sub_helper ./pkg/sub_parser_hub ./pkg/logic/mark_system ./pkg/downloader ./pkg/save_sub_helper ./pkg/settings ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier ./pkg/logic/sub_supplier/subhd -count=1`
- any additional targeted package tests required by touched files
- `npm run build` in `frontend`

### 2. Candidate Image Gate

- Build exactly one active candidate image:
  - `docker build --build-arg INSTALL_BROWSER=true -t chinesesubfinder:local-candidate .`
- Record image id and build time in the round report

### 3. Local Container Gate

Run exactly one active validation container from the candidate image, mounted
only to the isolated root:

- config root: `D:\tmp\csf-local-candidate\config`
- media root: `D:\tmp\csf-local-candidate\media`
- temp root: `D:\tmp\csf-local-candidate\tmp`
- optional artifacts root: `D:\tmp\csf-local-candidate\artifacts`

### 4. End-to-End Validation Matrix

The local validation matrix must cover these behaviors:

1. Chinese subtitle native success path
2. `subhd` path with `SVG direct read -> local ddddocr`
3. `SubtitleCat` English fallback default-enabled behavior
4. `SubtitleCat` translated Chinese fallback explicit switch behavior
5. LLM English-to-Chinese fallback behavior
6. Frontend settings visibility for:
   - built-in Chrome default behavior in Docker
   - `SubtitleCat` Chinese translated fallback switch
   - no misleading runtime-path requirements in Docker mode

The repeatable driver for the default matrix is:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_full_acceptance.ps1 `
  -LLMProvider deepseek `
  -LLMBaseUrl https://api.deepseek.com `
  -LLMModel deepseek-v4-flash `
  -LLMApiKey <current-session-key>
```

This driver runs all intended route phases against the same active
image/container/root and the real mounted library at
`\\192.168.100.4\video\link`:

1. movie native Chinese round
2. movie explicit SubtitleCat translated Chinese round
3. movie English fallback round
4. movie safe-fail round
5. series native Chinese round
6. series English fallback round
7. series explicit SubtitleCat translated Chinese round
8. movie LLM english-to-chinese fallback round
9. series LLM english-to-chinese fallback round

Current runtime note on 2026-06-20:

- on this machine, the raw direct FnOS sample pool is no longer a dependable
  Docker regression surface because host-side sample existence does not imply
  Docker visibility
- the acceptance entrypoints now auto-audit the requested sample pool first
- if the requested pool is not fully Docker-visible, they automatically reuse or
  materialize `D:\tmp\csf-real-media-runtime\sample-specs` from the same FnOS
  titles before running the matrix
- this keeps the config/auth/browser state on the pulled FnOS working volumes
  while normalizing the media surface into a Docker-readable local runtime pool

Important operational note:

- the full driver is still the authoritative whole-chain batch
- `subhd` daily download limits can block the first native-Chinese round before
  the batch reaches the LLM rounds
- because of that, LLM fallback now has a dedicated revalidation entrypoint on
  the same candidate/container/root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_llm_acceptance.ps1 `
  -LLMProvider deepseek `
  -LLMBaseUrl https://api.deepseek.com `
  -LLMModel deepseek-v4-flash `
  -LLMApiKey <current-session-key>
```

For broader low-cost real-library coverage without spending LLM tokens, use the
expanded non-LLM batch:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_expanded_acceptance.ps1
```

This expanded batch is intentionally sequential because all rounds mutate the
same candidate container and auth/session state. Parallel execution can trigger
false `AccessToken Error` failures in the harness.

Current harness note on 2026-06-20:

- `scripts/local_e2e_matrix.ps1` now enforces a process mutex for one
  `CandidateContainer + BaseUrl + ConfigDockerVolume` tuple
- failing isolated rounds restore from the stable local snapshot at
  `D:\tmp\csf-local-candidate\artifacts\baseline-settings.json` before the
  script rethrows the terminal error

To audit the real sample spec pool itself against the mounted UNC library and
the Docker bind-source mapping, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_sample_pool_audit.ps1
```

This audit must confirm that every JSON spec still resolves to a real `.mkv`
under `\\192.168.100.4\video\link` and that the pool stays on one movie root,
one series root, one Docker movie source, and one Docker series source.

To snapshot the currently retained route coverage window itself, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_route_coverage_snapshot.ps1
```

This snapshot must show all 9 required routes present before the current local
acceptance window can be treated as complete enough for release gating.

All real-library intended-chain rounds must use `-ExpectedRouteKey` and pass
the route assertion before the evidence is accepted. A generic successful job
is not enough if the supplier chain silently degraded to a different route.

Explicit single-round debugging is still allowed with
`scripts\local_candidate_round.ps1`.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_candidate_round.ps1 `
  -RunE2EMatrix `
  -EnableSubtitleCatTranslatedChineseFallback
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_candidate_round.ps1 `
  -RunE2EMatrix `
  -EnableLLMFallback `
  -LLMProvider deepseek `
  -LLMBaseUrl https://api.deepseek.com `
  -LLMModel deepseek-v4-flash `
  -JobTimeoutSeconds 900
```

Do not put secrets into the plan file or any report. Pass them as
current-session command arguments only.

### 4.1 Current Verified Evidence

The current retained local evidence set proves these matrix items on the active
candidate:

1. Movie native Chinese route passed with assertion:
   - `D:\tmp\csf-local-candidate\reports\20260619-072406-849-e2e-matrix\e2e-summary.json`
2. Movie explicit SubtitleCat translated Chinese route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-072444-724-e2e-matrix\e2e-summary.json`
3. Movie English fallback route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-075032-805-e2e-matrix\e2e-summary.json`
4. Movie safe-fail route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-072626-440-e2e-matrix\e2e-summary.json`
5. Series native Chinese route passed with assertion:
   - `D:\tmp\csf-local-candidate\reports\20260619-075153-872-e2e-matrix\e2e-summary.json`
6. Series English fallback route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-075055-866-e2e-matrix\e2e-summary.json`
7. Series explicit SubtitleCat translated Chinese route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-072827-356-e2e-matrix\e2e-summary.json`
8. Movie LLM fallback route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-084650-122-e2e-matrix\e2e-summary.json`
9. Series LLM fallback route passed:
   - `D:\tmp\csf-local-candidate\reports\20260619-090711-552-e2e-matrix\e2e-summary.json`

The real-title round is the authoritative proof for:

- Chinese subtitle native success path
- `subhd` live supplier path
- `SVG direct read -> local ddddocr` capable gate path without enabling any
  external OCR fallback by default

### 5. Residue Gate

Each round must classify residue into:

- keep for next round
- delete immediately after verification
- unexpected residue that indicates a broken harness

Residue classes to inspect:

- local containers
- local images
- tar archives
- `Once-*.log`
- generated subtitles
- temporary prompts/responses/debug artifacts
- browser cache under the test root

Cleanup is preview-only unless explicitly executed:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_residue_audit.ps1 `
  -WriteCleanupManifest
```

Then preview the current deletion set:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_cleanup.ps1
```

After explicit confirmation, execute:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\local_cleanup.ps1 -Execute
```

## Naming Rules

- Active candidate image tag: `chinesesubfinder:local-candidate`
- Active container name: `chinesesubfinder-local-candidate`
- Active test root: `D:\tmp\csf-local-candidate`
- Per-round report dir:
  - `D:\tmp\csf-local-candidate\reports\<timestamp>`

## Stop Conditions

Stop and fix locally before any release/FnOS work if any of these happen:

- more than one active candidate container exists
- more than one active candidate image tag is introduced for the same round
- residue report shows outputs outside the isolated root
- a required matrix item is untested
- frontend package and runtime package are not the same candidate build

## Release Entry Criteria

Do not enter GitHub/release/FnOS validation until all of the following are
true:

1. local test matrix completed
2. all required commands passed
3. residue report reviewed
4. only expected keep-set remains locally
5. current repo changes are understood and attributable to the tested
   candidate

## Current Keep Set

After the current local acceptance run, the only allowed keep-set is:

- active container: `chinesesubfinder-local-candidate`
- active image: `chinesesubfinder:local-candidate`
- active root: `D:\tmp\csf-local-candidate`
- sample pool: `D:\tmp\csf-real-media-stage`
- latest sample spec pool audit evidence only
- latest valid route-coverage evidence only, not every historical sample run

Everything else under this test harness should be treated as residue and either
explained or removed.
