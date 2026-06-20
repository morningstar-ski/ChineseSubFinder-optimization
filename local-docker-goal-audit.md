# Local Docker Goal Audit

Last updated: 2026-06-20 17:02 Asia/Shanghai

## Objective

Continue rebuilding and validating the local Docker acceptance loop for
`ChineseSubFinder-provider-pack` under the real mounted library at
`\\192.168.100.4\video\link`, while keeping the test surface local-first,
evidence-driven, and cleaned as work progresses.

## Requirement Audit

### A. One candidate image only

Status: proved

Evidence:

- active image:
  - `chinesesubfinder:local-candidate`
- latest residue audit:
  - `D:\tmp\csf-local-candidate\reports\residue-audit-20260619-093316-197.json`

### B. One isolated local candidate root

Status: proved

Evidence:

- active root is `D:\tmp\csf-local-candidate`
- residue audit keeps that root and the sample spec pool only

### C. Real mounted library is used

Status: proved

Evidence:

- live UNC root:
  - `\\192.168.100.4\video\link`
- current sample spec pool:
  - `D:\tmp\csf-real-media-stage`
- latest sample spec pool audit:
  - `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260619-053556-615.json`
- latest route coverage snapshot:
  - `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260619-181953-522.json`

### D. Local-first debugging, no FnOS debug loop

Status: proved

Evidence:

- fixes and acceptance runs were executed by local PowerShell and local Docker
- FnOS was used as the mounted media source only

### E. Route coverage

Status: proved for the current retained window

Evidence:

- `movie.native_chinese`
  - `20260619-072406-849-e2e-matrix`
- `movie.subtitlecat_translated`
  - `20260619-072444-724-e2e-matrix`
- `movie.english_fallback`
  - `20260619-075032-805-e2e-matrix`
- `movie.safe_fail`
  - `20260619-072626-440-e2e-matrix`
- `series.native_chinese`
  - `20260619-075153-872-e2e-matrix`
- `series.english_fallback`
  - `20260619-075055-866-e2e-matrix`
- `series.subtitlecat_translated`
  - `20260619-072827-356-e2e-matrix`
- `movie.llm_fallback`
  - `20260619-084650-122-e2e-matrix`
- `series.llm_fallback`
  - `20260619-090711-552-e2e-matrix`
- machine snapshot:
  - `present_route_count=10`
  - `missing_required_route_count=0`
  - `coverage_ok=true`

### F. SubHD local OCR / SVG chain

Status: proved

Evidence:

- current native movie proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-072406-849-e2e-matrix`
- current native series proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-075153-872-e2e-matrix`
- logs continue to show:
  - `subhd captcha using ocr backend: ddddocr`
  - `subhd download gate passed without captcha`
  - `subhd step0 keyword failed, continue next keyword ...`

### G. LLM fallback is decoupled from native-supplier daily limits

Status: improved and verified

Evidence:

- new entrypoint:
  - `scripts/local_llm_acceptance.ps1`
- shared profile source:
  - `scripts/local_acceptance_matrix.psd1`
- latest movie LLM proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-084650-122-e2e-matrix\e2e-summary.json`
- latest series LLM proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-090711-552-e2e-matrix\e2e-summary.json`
- both latest LLM summaries prove:
  - `job_terminal_status=3`
  - `final_output_has_chinese=true`
  - `route_assertion=passed`

### H. Cleanup is integrated into the flow

Status: proved

Evidence:

- latest residue audits retained:
  - `D:\tmp\csf-local-candidate\reports\residue-audit-20260619-093316-197.json`
  - `D:\tmp\csf-local-candidate\reports\residue-audit-20260619-092744-248.json`
- latest route coverage snapshot retained:
  - `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260619-092334-424.json`
- latest sample pool audit retained:
  - `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260619-053556-615.json`

Verified cleanup behavior:

- keep latest valid route-coverage evidence only
- keep one latest route coverage snapshot
- prune stale duplicate proofs after newer retained evidence lands
- remove failed duplicate windows once newer retained evidence exists
- preserve the sample spec pool and active candidate root

### J. Pulled FnOS config remains usable in local Docker

Status: improved and re-verified

Evidence:

- live FnOS config repull helper:
  - `scripts/pull_fnos_full_config.ps1`
- latest repull manifest:
  - `D:\tmp\fnos-csf-config-pull-20260619-194339.json`
- current live FnOS reuse source:
  - `fnos-csf:/vol1/1000/docker/csf/config`
  - `fnos-csf:/vol1/1000/docker/csf/browser`
- immutable pulled baseline is now preserved as local Docker volumes:
  - `csf_fnos_config_full_20260619`
  - `csf_fnos_browser_full_20260619`
- actual test rounds now run on cloned clean working volumes:
  - `csf_fnos_config_working`
  - `csf_fnos_browser_working`
- refresh helper:
  - `scripts/refresh_fnos_working_volumes.ps1`
- latest working-volume proofs:
  - `assrt`:
    - `D:\tmp\csf-local-candidate\reports\20260619-153813-522-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260619-165256-310-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260619-200709-664-e2e-matrix`
  - `opensubtitles`:
    - `D:\tmp\csf-local-candidate\reports\20260619-154236-887-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260619-164307-305-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260619-195837-520-e2e-matrix`
  - `subhd`:
    - `D:\tmp\csf-local-candidate\reports\20260619-154351-895-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260619-201239-517-e2e-matrix`
  - `subdl`:
    - `D:\tmp\csf-local-candidate\reports\20260619-201123-934-e2e-matrix`
- these summaries prove:
  - `job_terminal_status=3`
  - `final_output_has_chinese=true` for native-Chinese proofs
  - `route_key=movie.english_fallback`, `final_output_has_chinese=false` for
    the latest isolated `opensubtitles` English-fallback proof
  - the latest isolated `subdl` proof is no longer just a config guess:
    the real pulled key was used in the request, but this sampled episode still
    ended at `job_terminal_status=2` with `No Sub Found`
- harness fixes now also prove:
  - the candidate container must be started with the same real-media mount
    sources that the e2e sample spec expects
  - route-setting updates must be derived from the local full-config shadow
    copy instead of a redacted `GET /v1/settings` response, or provider tokens
    are lost during tests
  - the latest repull re-established parity between live FnOS settings and the
    immutable local baseline before the working volumes were refreshed
  - after each repull, the working volume still needs the local runtime
    adaptation step so the pulled supplier settings are preserved while the
    actual scan roots are rewritten back to `/media/movies` and
    `/media/series`
  - route isolation must not clear supplier secrets when a supplier is merely
    disabled for a single round; the harness now preserves those credentials
    and relies on supplier enable flags plus daily-limit gates for isolation

### K. Timeline fixing no longer blocks long-movie completion

Status: improved and verified

Evidence:

- code change:
  - `pkg/logic/sub_timeline_fixer/SubTimelineFixerHelperEx.go`
- focused unit proof:
  - `go test ./pkg/logic/sub_timeline_fixer -run TestShouldSkipAudioFallbackTimelineFix -count=1`
- live runtime proof from container logs:
  - `Skip TimeLine Fix -- audio fallback duration too long: 10144.2177734375 ...`
- downstream acceptance proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-140132-637-e2e-matrix`
  - `D:\tmp\csf-local-candidate\reports\20260619-140458-527-e2e-matrix`

### L. SubDL policy still needs a final route decision

Status: narrowed and still incomplete

Evidence:

- route-classification fix:
  - `scripts/local_e2e_matrix.ps1`
  - `AcceptNoSubFound` no longer forces all successful outputs to be labeled
    `safe_fail`
- current serial movie proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-181417-903-e2e-matrix`
  - proves:
    - `job_terminal_status=3`
    - `route_key=movie.english_fallback`
    - `final_output_has_chinese=false`
- current serial series proof:
  - `D:\tmp\csf-local-candidate\reports\20260619-180008-382-e2e-matrix`
  - proves:
    - `job_terminal_status=3`
    - `route_key=series.english_fallback`
    - `final_output_has_chinese=false`
- current conclusion:
  - `subdl` does not currently justify a broad native-Chinese-primary role
    under the sampled real-library evidence window
  - current sampled movie and series behavior both fall through to
    English-only output on the same supplier

### M. Credential-preserving isolation fix is now live

Status: improved and verified

Evidence:

- code fix:
  - `scripts/local_e2e_matrix.ps1`
  - the harness no longer blanks `assrt` / `subdl` credentials when those
    suppliers are disabled for the current round
- recovery step:
  - `scripts/refresh_fnos_working_volumes.ps1`
- post-fix revalidation:
  - `D:\tmp\csf-local-candidate\reports\20260619-180927-366-e2e-matrix`
  - proves:
    - `job_terminal_status=3`
    - `route_key=series.native_chinese`
    - `final_output_has_chinese=true`
- post-fix isolation proof on a `subdl`-only movie round:
  - `D:\tmp\csf-local-candidate\reports\20260619-181417-903-e2e-matrix`
  - proves:
    - `job_terminal_status=3`
    - `route_key=movie.english_fallback`
    - supplier credentials in the working volume still remain present after
      the round
- post-run working-volume check:
  - `assrt_token_present=true`
  - `subdl_key_present=true`
  - `opensub_api_present=true`

### N. Default chain layering is now cleaner

Status: improved and verified

Evidence:

### O. Failure-path restore of pulled FnOS working config

Status: proved

Evidence:

- harness fixes:
  - `scripts/local_e2e_matrix.ps1`
  - `scripts/local_candidate_round.ps1`
- stable local baseline snapshot:
  - `D:\tmp\csf-local-candidate\artifacts\baseline-settings.json`
- isolated failure proof:
  - `D:\tmp\csf-local-candidate\reports\20260620-040604-670-e2e-matrix\failure.json`
  - proves:
    - `status=2`
    - `error=No Sub Found`
- post-failure provider-state proof:
  - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-040650-854\summary.json`
  - proves:
    - `assrt valid=true`
    - `subdl valid=true`
    - `opensubtitles valid=true`
    - `tvsubtitles valid=true`
    - `moviesubtitles valid=true`
    - `subtitlecat valid=true`
    - `subhd valid=true`
    - `subtitle_best valid=false reason=disabled`

- code changes:
  - `pkg/logic/pre_download_process/pre_download_proces.go`
  - `pkg/types/common/sub_site_sequence.go`
- unit coverage:
  - `go test ./pkg/logic/pre_download_process -count=1`
  - `go test ./pkg/types/common -count=1`
  - `go test ./pkg/downloader -count=1`
- policy now enforced in code:
- `subdl` is English-fallback-only by default
- `tvsubtitles` is no longer part of the default fallback chain
- `moviesubtitles` is English-fallback-only by default
- `opensubtitles` stays available in both native and English fallback stages
- `subtitlecat` stays out of the native primary chain and remains a fallback
- rebuilt local candidate proof window:
  - native Chinese:
    - `D:\tmp\csf-local-candidate\reports\20260619-183251-124-e2e-matrix`
    - proves `route_key=series.native_chinese`
  - mixed English fallback movie:
    - `D:\tmp\csf-local-candidate\reports\20260619-183355-308-e2e-matrix`
    - proves `route_key=movie.english_fallback`
  - mixed English fallback series:
    - `D:\tmp\csf-local-candidate\reports\20260619-183500-434-e2e-matrix`
    - proves `route_key=series.english_fallback`
  - clean safe-fail movie:
    - `D:\tmp\csf-local-candidate\reports\20260619-181812-104-e2e-matrix`
    - proves `route_key=movie.safe_fail`

### I. Script structure is less drift-prone

Status: improved and verified

Evidence:

- shared batch definitions:
  - `scripts/local_acceptance_matrix.psd1`
- shared batch executor:
  - `scripts/local_acceptance_runner.ps1`
- verified entrypoints:
  - `scripts/local_full_acceptance.ps1`
  - `scripts/local_expanded_acceptance.ps1`
  - `scripts/local_llm_acceptance.ps1`
- current route-isolation hardening:
  - `scripts/local_e2e_matrix.ps1`
  - explicit `subtitlecat_translated` rounds now isolate SubtitleCat by
    default when no supplier set is explicitly requested, preventing
    false-positive route classification from other Chinese suppliers
  - single-supplier isolation rounds now also assert the actual winning
    supplier from container logs through `supplier-evidence.json`, rather than
    inferring the route only from output language
  - latest verified supplier-assertion proofs:
    - `D:\tmp\csf-local-candidate\reports\20260619-212036-542-e2e-matrix`
      proves `actual_supplier=subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260619-211716-624-e2e-matrix`
      proves `actual_supplier=subtitlecat`

## Current Conclusion

The local acceptance loop is materially stronger and cleaner than the earlier
state.

What is currently solid:

- real mounted movie and series paths under `\\192.168.100.4\video\link`
- immutable pulled FnOS baseline plus refreshable clean working volumes for
  actual local testing
- repeatable live-config repull from the real FnOS CSF config directories into
  the immutable local baseline volumes
- fail-fast mounted-path validation in `scripts/local_e2e_matrix.ps1`, so
  sample/container mount mismatches no longer masquerade as random job timeouts
- live `subhd` OCR path with `ddddocr` as the main local OCR
- live `assrt` movie native-Chinese proof on clean working volume
- live `assrt` series native-Chinese proof on clean working volume
- live `opensubtitles` movie native-Chinese proof on clean working volume
- `SubtitleCat` English fallback route on movie and series samples
- `SubtitleCat` explicit translated-Chinese route on movie and series samples
- fresh movie and series LLM fallback proofs under the newest matrix-driven
  harness
- route coverage is machine-snapshotted instead of being reconstructed by hand
- standard, expanded, and LLM-only batches are matrix-driven instead of
  duplicating route definitions across multiple scripts
- local residue audit and cleanup hold the keep-set to one active candidate
  image/container plus the current evidence window
- the latest isolated `opensubtitles` proof under the re-pulled FnOS config
  also ended with sample-library cleanup, with the generated subtitle residue
  removed from `记忆碎片 (2000)` after evidence capture
- the latest isolated `assrt` and `subhd` proofs under the re-pulled FnOS
  config also ended with sample-library cleanup, with the generated `.csf-bk`
  residues removed from `黑袍纠察队 (2019) S01E03` after evidence capture
- the latest isolated `moviesubtitles` regression proof under the re-pulled
  FnOS config also ended cleanly, with no subtitle residue left behind in
  `沉默的羔羊 (1991)` after evidence capture
- the latest isolated `subtitlecat_translated` and default `subtitlecat`
  proofs under the re-pulled FnOS config also ended with sample-library
  cleanup, with the generated `.csf-bk` residues removed from
  `洛佩兹一家 (2002) S01E01` after evidence capture

Still not enough to mark the overall goal complete:

- `subdl` still needs a route-policy decision because the sampled real-library
  evidence now splits into `movie.safe_fail` and `series.english_fallback`

### N. 2026-06-20 supplier follow-up tightened the remaining route decisions

Status: narrowed further

Evidence:

- isolated `tvsubtitles` rerun with primary Chinese suppliers explicitly
  disabled:
  - `D:\tmp\csf-local-candidate\reports\20260620-004014-640-e2e-matrix`
  - proves:
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
    - `route_key=series.safe_fail`
- isolated `subdl` series rerun with primary Chinese suppliers explicitly
  disabled:
  - `D:\tmp\csf-local-candidate\reports\20260620-004142-371-e2e-matrix`
  - proves:
    - `actual_supplier=subdl`
    - `route_key=series.english_fallback`
- isolated `subdl` movie rerun with primary Chinese suppliers explicitly
  disabled:
  - `D:\tmp\csf-local-candidate\reports\20260620-004512-354-e2e-matrix`
  - proves:
    - `actual_supplier=subdl`
    - `route_key=movie.english_fallback`
- code inspection also settled the remaining `subtitle_best` ambiguity:
  - supplier implementation:
    - `pkg/logic/sub_supplier/subtitle_best/subtitle_best.go`
    - `pkg/logic/sub_supplier/subtitle_best/api.go`
  - auxiliary shared service usage:
    - `pkg/subtitle_best_api/subtitle_best_api.go`
    - `pkg/media_info_dealers/dealers.go`
    - `pkg/logic/pre_download_process/pre_download_proces.go`
  - conclusion:
    - `subtitle_best` is not just an old supplier toggle; it also backs shared
      `subhd` code fetches and metadata / ID-conversion fallback

Current conclusion:

- `subdl` is now proven as a working English-fallback supplier for both movie
  and series samples, but it still does not justify any native-Chinese-primary
  role
- `tvsubtitles` now has a true isolated failure proof on the current runtime
  and no longer justifies remaining in the default English fallback chain
- `subtitle_best` still cannot be marked runtime-verified as a supplier because
  the pulled FnOS config keeps its API key empty, but its shared-service role is
  real and still relevant
- broader low-cost title sampling is still needed if a wider confidence window
  is required

Addendum on 2026-06-20 after rebuilt-image reruns:

- the above route decision is now also re-proven on the current candidate image
  mounted against freshly staged real-library samples:
  - `subdl`:
    - `D:\tmp\csf-local-candidate\reports\20260620-010357-745-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_key=series.english_fallback`
  - `tvsubtitles`:
    - `D:\tmp\csf-local-candidate\reports\20260620-010702-382-e2e-matrix`
    - `job_error_info=No Sub Found`
    - `route_key=series.safe_fail`
  - `subtitlecat`:
    - `D:\tmp\csf-local-candidate\reports\20260620-010755-329-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_key=series.english_fallback`
  - `subtitlecat_translated`:
    - `D:\tmp\csf-local-candidate\reports\20260620-010849-101-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_key=series.subtitlecat_translated`
- this closes the remaining ambiguity about whether the latest policy change
  accidentally broke the working English or explicit translated fallback paths:
  it did not

### O. 2026-06-20 local Docker direct-mount reality check and runtime-pool fix

Status: improved

Evidence:

- the strengthened sample-pool audit now checks Docker visibility in addition to
  host-side existence:
  - direct real-library pool:
    `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260620-012836-871.json`
    - proves `existing_host_video_count=10`
    - proves `docker_visible_video_count=0`
  - therefore the current machine cannot treat the raw FnOS share as a reliable
    active Docker bind source for the intended regression samples, even after
    correcting the old `/run/desktop/mnt/fnos-real/...` bridge path
- a dedicated materialize helper now converts that real-library sample pool
  into a minimized Docker-readable local runtime pool while preserving the same
  titles and same pulled FnOS config:
  - `scripts/local_materialize_sample_pool.ps1`
  - output:
    `D:\tmp\csf-real-media-runtime`
  - report:
    `D:\tmp\csf-real-media-runtime\materialize-report.json`
- the runtime pool passed the new stronger audit:
  - `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260620-020123-986.json`
  - proves `docker_visible_video_count=10`
- the expanded acceptance entrypoints were updated to pass through
  `ConfigDockerVolume` / `BrowserDockerVolume`, and the expanded profile now
  starts the container on its first round
- with that runtime pool and the live pulled FnOS working volumes, the latest
  expanded acceptance reruns passed:
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-020331-265-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-020423-943-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-020459-622-e2e-matrix`
    - `actual_supplier=subhd`

Current conclusion:

- on this machine, the real FnOS share remains the authoritative source of test
  media, but raw direct Docker mounts cannot currently be treated as a stable
  regression surface for the intended sample pool
- the materialized runtime pool is now the reliable local-Docker surrogate for
  those same real-library titles until the host-side Docker bridge issue is
  solved
- the current intended non-LLM chain is still passing after the latest route
  policy changes when exercised against that normalized runtime pool

### P. 2026-06-20 auto-resolved runtime pool now covers the whole intended chain

Status: improved

Evidence:

- acceptance entrypoints now auto-resolve from the requested real-library stage
  pool into a Docker-readable runtime pool when needed
- LLM refresh on the same auto-resolved runtime pool passed:
  - `movie.llm_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-022515-500-e2e-matrix`
    - `route_assertion=passed`
    - `actual_supplier=subtitlecat`
  - `series.llm_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-024514-233-e2e-matrix`
    - `route_assertion=passed`
    - `actual_supplier=subtitlecat`
- full non-LLM acceptance refresh on the same auto-resolved runtime pool passed:
  - `movie.native_chinese`
    - `20260620-025717-754-e2e-matrix`
    - `actual_supplier=subhd`
  - `movie.subtitlecat_translated`
    - `20260620-025816-538-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
  - `movie.english_fallback`
    - `20260620-030020-050-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `movie.safe_fail`
    - `20260620-030058-248-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `20260620-030132-231-e2e-matrix`
    - `actual_supplier=subhd`
  - `series.english_fallback`
    - `20260620-030243-315-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.subtitlecat_translated`
    - `20260620-030453-832-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`

Current conclusion:

- the current local-Docker acceptance flow now no longer depends on a manually
  prepared per-title staged directory
- it can start from the original real-library sample pool request, detect when
  raw direct Docker visibility is broken, and continue on the normalized runtime
  pool without losing the same titles or the pulled FnOS config/browser state
- the present remaining uncertainty is no longer the non-LLM or LLM route logic
  itself; it is mainly whether the raw direct FnOS share can later be restored
  as a first-class Docker mount surface on this host

Addendum on 2026-06-20 after harness hardening:

- refreshed working volumes no longer require a container restart just to make
  the config readable again:
  - `scripts/refresh_fnos_working_volumes.ps1` now reapplies ownership for the
    container runtime user
  - verification:
    - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-044041-605`
    - `subdl valid=true` immediately after refresh
- the local runtime adaptation step in `scripts/local_candidate_round.ps1` no
  longer leaks ordered-dictionary meta fields such as `Count`, `Keys`, and
  `Values` into `config-shadow\ChineseSubFinderSettings.json`
  - this was the root cause behind a false harness failure where
    `subtitlecat_settings.enabled` disappeared from the adapted shadow config
- the matrix harness now makes the `tvsubtitles` policy mismatch explicit:
  - `D:\tmp\csf-local-candidate\reports\20260620-044610-120-e2e-matrix`
  - `policy_warnings.json` now states that `tvsubtitles` is not wired into the
    backend default English fallback chain
  - the same round still ends with the real isolated runtime result:
    `job_error_info=No Sub Found`

Addendum on 2026-06-20 after the next full non-LLM acceptance rerun:

- the latest non-LLM full acceptance pass re-ran cleanly on candidate image
  `cd2dd11cfc8f` after the ownership and config-merge fixes
- current full-route proof set from that rerun:
  - `movie.native_chinese`
    - `20260620-045337-821-e2e-matrix`
    - `actual_supplier=subhd`
  - `movie.subtitlecat_translated`
    - `20260620-045436-321-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
  - `movie.english_fallback`
    - `20260620-045708-901-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `movie.safe_fail`
    - `20260620-045800-023-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `20260620-045838-216-e2e-matrix`
    - `actual_supplier=subhd`
  - `series.english_fallback`
    - `20260620-050123-229-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.subtitlecat_translated`
    - `20260620-050358-181-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
- route coverage snapshot after that rerun is now fully green:
  - `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260620-050835-197.json`
  - `coverage_ok=true`
  - `missing_required_route_count=0`
- cleanup was executed immediately after the rerun:
  - stale duplicate report directories and old snapshot artifacts were deleted
  - current residue state:
    - `D:\tmp\csf-local-candidate\reports\residue-audit-20260620-050909-387.json`
    - one active candidate container
    - one active candidate image
    - no extra csf temp roots outside the active candidate root and the two
      sample pools

Addendum on 2026-06-20 after default all-supplier reality-check rounds:

- the latest default-state movie round, without narrowing suppliers, still
  resolved cleanly to the intended Chinese-first path:
  - `D:\tmp\csf-local-candidate\reports\20260620-051046-861-e2e-matrix`
  - `route_key=movie.native_chinese`
  - `actual_supplier=subhd`
- the latest default-state series round, also without narrowing suppliers,
  resolved to a different but valid native-Chinese winner:
  - `D:\tmp\csf-local-candidate\reports\20260620-051237-227-e2e-matrix`
  - `route_key=series.native_chinese`
  - `actual_supplier=opensubtitles`
- this confirms the current macro behavior is healthier than a single-source
  funnel:
  - the Chinese-first stage can still prefer `subhd` for some titles
  - but the same stage is able to let `opensubtitles` win when it produces the
    better usable Chinese subtitle
- latest supplier snapshot on that same default runtime:
  - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-051618-724`
  - valid:
    `xunlei`, `shooter`, `assrt`, `subdl`, `opensubtitles`, `tvsubtitles`,
    `moviesubtitles`, `subtitlecat`, `subhd`
  - invalid:
    `subtitle_best` with `reason=disabled`

Addendum on 2026-06-20 after route-stage evidence hardening:

- the local candidate image was rebuilt as `c946b859fc90`
- the downloader now emits explicit `SubtitleRouteStage` logs for successful
  `primary_chinese`, `translated_chinese`, `llm_fallback`,
  `english_fallback`, and terminal `safe_fail`
- `scripts/local_e2e_matrix.ps1` now captures those logs into
  `supplier-evidence.json` and uses them to derive `route_key`, instead of
  relying only on whether the saved subtitle sample text happens to contain
  Chinese
- rebuilt-image proof:
  - `subdl` isolated movie round:
    - `D:\tmp\csf-local-candidate\reports\20260620-054350-148-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=movie.english_fallback`
  - `opensubtitles` isolated movie round:
    - `D:\tmp\csf-local-candidate\reports\20260620-054437-964-e2e-matrix`
    - `actual_supplier=opensubtitles`
    - `route_stage=primary_chinese`
    - `route_key=movie.native_chinese`
- this matters because it converts a previous gray area into a hard finding:
  - `opensubtitles` still participates in the backend primary Chinese chain
    whenever it is enabled
  - therefore it cannot be isolated as an English-only fallback supplier under
    the current runtime policy
- the harness now surfaces that coupling as an explicit warning:
  - `D:\tmp\csf-local-candidate\reports\20260620-054642-873-e2e-matrix`
  - `policy_warnings.json` now states that requested English fallback supplier
    `opensubtitles` also participates in the backend primary Chinese chain when
    enabled

Addendum on 2026-06-20 after explicit stage-order refactor and current-image reruns:

- the fallback-chain architecture was tightened without changing intended
  runtime policy:
  - primary Chinese supplier order is now explicit
  - default English fallback supplier order is now explicit
  - translated-Chinese fallback order is now explicit
- implementation:
  - `pkg/types/common/sub_site_sequence.go`
  - `pkg/logic/pre_download_process/pre_download_proces.go`
- regression coverage was added and passed:
  - `pkg/types/common/sub_site_sequence_test.go`
  - `pkg/logic/pre_download_process/pre_download_proces_test.go`
  - `go test ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/downloader -count=1`
- after that refactor, the full local non-LLM acceptance suite re-passed again
  on the current active image `beb997acf572`
  - `movie.native_chinese`
    - `20260620-061444-480-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `movie.subtitlecat_translated`
    - `20260620-062033-929-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
  - `movie.english_fallback`
    - `20260620-062210-684-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_stage=english_fallback`
  - `movie.safe_fail`
    - `20260620-062243-708-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `20260620-062311-276-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `series.english_fallback`
    - `20260620-062409-742-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_stage=english_fallback`
  - `series.subtitlecat_translated`
    - `20260620-062448-272-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
- default English fallback competition was then re-checked on the same active
  image:
  - movie competition among `subdl`, `subtitlecat`, and `moviesubtitles`:
    - `20260620-062835-621-e2e-matrix`
    - winner: `subdl`
  - series competition among `subdl` and `subtitlecat`:
    - `20260620-062924-525-e2e-matrix`
    - winner: `subdl`
  - movie-only tail fallback proof:
    - `20260620-060341-454-e2e-matrix`
    - winner: `moviesubtitles`
- current policy conclusion:
  - keep `subdl` ahead of `subtitlecat` in the default English fallback chain
  - keep `subtitlecat` as the broader downstream fallback
  - keep `moviesubtitles` as the movie-only tail fallback
  - keep `opensubtitles` out of English-only isolation claims because it still
    participates in the primary Chinese chain when enabled

Addendum on 2026-06-20 after focused supplier re-probe and report hardening:

- `subdl` has now been re-proved again under true primary-chain isolation on
  the current active image:
  - movie:
    - `20260620-063728-937-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
  - series:
    - `20260620-063804-363-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
- `tvsubtitles` was isolated again and still does not justify any default-chain
  role:
  - `20260620-063847-623-e2e-matrix`
  - hard findings:
    - `policy_warnings.json` says `tvsubtitles` is not wired into the backend
      default English fallback chain
    - runtime result stayed `No Sub Found`
- `subtitle_best` ambiguity is now fully reduced to one remaining point:
  - confirmed by code:
    - it is both a supplier implementation and a shared subtitle.best-backed
      support service for `subhd` code fetch and media-info / id-convert
      fallback
  - still not runtime-proved as a supplier because the active pulled FnOS
    working config keeps it disabled with an empty API key
- the local acceptance artifact layer was hardened so malformed sample-name
  strings no longer break JSON reports:
  - shared helper added in `scripts/local_acceptance_matrix_utils.ps1`
  - writer call sites switched in the local acceptance / sample-pool / supplier
    snapshot scripts
  - post-fix live parseable reports:
    - `20260620-064523-788-e2e-matrix`
    - `20260620-064711-526-e2e-matrix`
- current status board for these three decision points:
  - confirmed usable:
    - `subdl` as default English fallback, ahead of `subtitlecat`
  - confirmed not suitable for default chain:
    - `tvsubtitles`
  - still awaiting supplier-level runtime proof:
    - `subtitle_best`

Addendum on 2026-06-20 after acceptance-matrix contract tightening and full rerun:

- default-English-fallback ordering is no longer just a manual interpretation of
  old reports; it is now encoded as an acceptance contract:
  - `scripts/local_acceptance_matrix.psd1`
  - `scripts/local_acceptance_runner.ps1`
  - `scripts/local_candidate_round.ps1`
  - `scripts/local_e2e_matrix.ps1`
- the matrix now asserts:
  - movie default English fallback winner must be `subdl` when
    `subdl + subtitlecat + moviesubtitles` compete
  - series default English fallback winner must be `subdl` when
    `subdl + subtitlecat` compete
- direct proof rounds for that contract:
  - movie: `20260620-065216-314-e2e-matrix`
  - series: `20260620-065216-762-e2e-matrix`
- after that change, the full local non-LLM acceptance suite re-passed again on
  rebuilt image `2da7ffb59123`
  - `movie.native_chinese`
    - `20260620-065900-907-e2e-matrix`
    - winner: `subhd`
  - `movie.subtitlecat_translated`
    - `20260620-070004-943-e2e-matrix`
    - winner: `subtitlecat_translated`
  - `movie.english_fallback`
    - `20260620-070139-977-e2e-matrix`
    - winner: `subdl`
  - `movie.safe_fail`
    - `20260620-070209-868-e2e-matrix`
    - `No Sub Found`
  - `series.native_chinese`
    - `20260620-070238-997-e2e-matrix`
    - winner: `subhd`
  - `series.english_fallback`
    - `20260620-070354-212-e2e-matrix`
    - winner: `subdl`
  - `series.subtitlecat_translated`
    - `20260620-070433-659-e2e-matrix`
    - winner: `subtitlecat_translated`
- targeted Go verification and frontend production build also passed in the same
  rerun:
  - `20260620-065430-096\go-targeted-tests.log`
  - `20260620-065430-096\frontend-build.log`
- net effect:
  - the default fallback-chain strategy is now both architecturally explicit and
    acceptance-enforced, not just socially documented

Addendum on 2026-06-20 after subtitle content sample audit:

- a representative manual content spot-check was added on top of route/supplier
  assertions using the latest non-LLM rerun outputs:
  - `20260620-065900-907-e2e-matrix` (`subhd`, movie native Chinese)
  - `20260620-070004-943-e2e-matrix` (`subtitlecat_translated`, movie translated Chinese)
  - `20260620-070139-977-e2e-matrix` (`subdl`, movie English fallback)
  - `20260620-070238-997-e2e-matrix` (`subhd`, series native Chinese)
  - `20260620-070354-212-e2e-matrix` (`subdl`, series English fallback)
  - `20260620-070433-659-e2e-matrix` (`subtitlecat_translated`, series translated Chinese)
- what this additional audit now proves:
  - sampled successful rounds are not just "downloaded a file"; they produced
    plausible subtitle bodies that match the intended route/language stage
  - no obvious wrong-title attach or empty-body false success was seen in the
    sampled outputs
- remaining limitation:
  - this is still a spot-check layer, not a statistically significant semantic
    accuracy benchmark
  - translated-Chinese fallback remains clearly weaker in fluency/polish than
    native-Chinese source hits

Addendum on 2026-06-20 after live supplier snapshot recheck:

- a fresh supplier snapshot was taken from the active local candidate runtime:
  - `supplier-status-20260620-080359-173`
  - probe layer result:
    - `subdl valid=true`
    - `tvsubtitles valid=true`
    - `subtitlecat valid=true`
    - `subtitle_best valid=false reason=disabled`
- the local snapshot harness was then upgraded to separate "probe health" from
  "active route role":
  - `scripts/local_supplier_status_snapshot.ps1` now records:
    - `participates_in_primary_chain`
    - `participates_in_default_english_fallback`
    - `participates_in_translated_chinese_fallback`
    - `policy_state`
    - `policy_note`
  - this matters because `tvsubtitles valid=true` previously looked stronger
    than it really was
- a new isolated `tvsubtitles`-only round was run immediately after that live
  snapshot:
  - `20260620-080111-540-e2e-matrix`
  - outcome:
    - harness warning says `tvsubtitles` is not wired into the backend default
      English fallback chain
    - runtime still ended with `No Sub Found`
- net status after this recheck:
  - confirmed usable:
    - `subdl` as the default English fallback supplier
    - `subtitlecat` as the downstream English fallback and explicit translated
      Chinese fallback when that switch is enabled
  - confirmed probe-only / not default-chain:
    - `tvsubtitles`
  - still awaiting supplier-level runtime proof:
    - `subtitle_best`

Addendum on 2026-06-20 after LLM audit hardening and candidate-root cleanup:

- the default subtitle content audit now also covers:
  - `movie.llm_fallback`
  - `series.llm_fallback`
  - latest saved report:
    `subtitle-content-audit-20260620-081447-692.json`
- the audit now computes `english_line_ratio` and emits warnings for fallback
  outputs that still leak noticeable untranslated English
- verified current LLM route quality state:
  - movie LLM fallback:
    - `20260620-022515-500-e2e-matrix`
    - `route_stage=llm_fallback`
    - `31` English-only lines across `2936` dialogue lines
  - series LLM fallback:
    - `20260620-024514-233-e2e-matrix`
    - `route_stage=llm_fallback`
    - `59` English-only lines across `870` dialogue lines
  - interpretation:
    - LLM fallback is functionally working
    - but the retained series sample still leaks enough untranslated English
      that it should be treated as operational-yet-not-fully-polished
- cleanup rules were also tightened inside the active candidate root without
  touching live config files:
  - old task dirs under
    `config-prepull-snapshot\llm-logs\*` are now auditable and deletable
  - this round reduced retained LLM log task dirs from `8` to `2`
  - retained LLM debug-log size dropped from about `7.83 MB` to `2.31 MB`
- supporting verification:
  - `go test ./pkg/llm_subtitle_fallback ./pkg/downloader -count=1`
    passed after the audit-script changes

Addendum on 2026-06-20 after targeted LLM repair-pass verification:

- the subflow translation path was tightened so that:
  - obviously untranslated English dialogue cues get a targeted repair pass
  - post-processed empty noise cues no longer fall back to the original source
    English text
- static verification after that change passed:
  - `python -m py_compile third_party\subflow\src\subflow\translate_job.py`
  - `go test ./pkg/llm_subtitle_fallback ./pkg/downloader -count=1`
- two live rebuilt-image reruns then verified the same real-library LLM series
  sample again:
  - `20260620-082931-240-e2e-matrix`
    - `english_only_line_count=25 / dialogue_line_count=877`
  - `20260620-084436-792-e2e-matrix`
    - `english_only_line_count=16 / dialogue_line_count=863`
- this gives a concrete local improvement ladder for the retained series sample:
  - older retained baseline:
    - `59 / 870`
  - first repair pass:
    - `25 / 877`
  - after empty-fallback fix:
    - `16 / 863`
- the remaining English-only residue on the newest retained series sample is
  now dominated by standalone speaker labels instead of whole dialogue lines
- current LLM-chain status after this pass:
  - functional and materially cleaner than before
  - remaining polish target:
    - speaker-label normalization rather than general untranslated dialogue

Addendum on 2026-06-20 after speaker-label normalization follow-up:

- `third_party/subflow/src/subflow/translate_job.py` was tightened again so
  raw English speaker labels no longer count as acceptable residue:
  - bare English labels such as `[Matty]`
  - mixed label lines such as `[Rya] 我去了...`
- focused verification added:
  - `python -m unittest subflow.test_translate_job subflow.test_openai_compatible_client`
  - passed
- two more rebuilt-image real-library reruns then verified the same retained
  LLM series sample:
  - `20260620-090352-653-e2e-matrix`
    - `english_only_line_count=7 / dialogue_line_count=866`
  - `20260620-091857-671-e2e-matrix`
    - `english_only_line_count=4 / dialogue_line_count=867`
    - `mixed_language_line_count=22`
- updated local evidence ladder for the same sample is now:
  - `59 / 870`
  - `25 / 877`
  - `16 / 863`
  - `7 / 866`
  - `4 / 867`
- this materially changes the quality assessment:
  - the route is not merely operational
  - it is now substantially closer to a deliverable fallback on the retained
    series sample, with residual English concentrated in proper-name fragments
    rather than untranslated dialogue blocks

Addendum on 2026-06-20 after cross-sample LLM verification:

- the current FnOS-backed local Docker candidate was then tested on additional
  real mounted-library samples to make sure the recent LLM cleanup was not
  overfit to a single episode:
  - `the-boys-s01e03.json`
    - retained round: `20260620-093247-472-e2e-matrix`
    - `series.llm_fallback`
    - `english_only_line_count=8 / dialogue_line_count=1293`
    - no warning emitted
  - `interstellar-2014.json`
    - `20260620-094608-548-e2e-matrix`
      - `17 / 2939`
    - `20260620-101525-600-e2e-matrix`
      - `9 / 2938`
    - `20260620-104213-358-e2e-matrix`
      - `7 / 2942`
- the movie-side tightening came from two narrow LLM adjustments:
  - one-letter OCR fragments such as `S.` / `T.` / `A.` no longer count as
    acceptable English-only residue
  - standalone spoken call-names are now nudged toward Chinese transliteration
    when the local context makes that obvious
- newest retained movie residuals are now concentrated in:
  - `NASA`
  - `Lazarus`
  - `CASE`
  - `RPM`
  - `TARS`
- this is enough to sharpen the route-level status:
  - `subtitlecat -> llm_fallback` is now supported by multi-sample evidence on
    both series and movie workloads
  - the remaining work is no longer general route stability
  - it is final polish around which specialized names or sci-fi terms should be
    transliterated versus intentionally preserved

Addendum on 2026-06-20 after supplier-role recheck:

- a fresh live supplier snapshot on the current local Docker candidate showed:
  - valid:
    - `xunlei`
    - `shooter`
    - `assrt`
    - `subdl`
    - `opensubtitles`
    - `tvsubtitles`
    - `moviesubtitles`
    - `subtitlecat`
    - `subhd`
  - invalid / disabled:
    - `subtitle_best`
- the key new runtime-role clarifications are:
  - `opensubtitles`
    - cannot currently be isolated as English-only under this backend policy
    - new direct proof round:
      - `20260620-111139-680-e2e-matrix`
      - actual result:
        - `actual_supplier=opensubtitles`
        - `route_key=movie.native_chinese`
        - `route_stage=primary_chinese`
    - this proves it is a real active primary-chain supplier here, not merely a
      health-probe success
  - `moviesubtitles`
    - new isolated proof round:
      - `20260620-111318-314-e2e-matrix`
      - `actual_supplier=moviesubtitles`
      - `route_key=movie.english_fallback`
      - `route_stage=english_fallback`
      - `route_assertion=passed`
    - this proves the movie-tail English fallback path is not only theoretical
- cleanup policy was then corrected so that future routine cleanup keeps:
  - route-level acceptance evidence
  - latest distinct supplier-isolation evidence per route
  - two recent policy-warning rounds instead of one
- without that retention fix, the new `opensubtitles` and `moviesubtitles`
  proofs would have been lost even though they materially improved the
  verification surface

- a second retention gap was then fixed for safe-fail isolation rounds:
  - problem:
    - older `No Sub Found` rounds did not preserve enough request context in
      `e2e-summary.json`, so cleanup could not distinguish which isolated
      supplier had been tested
  - implementation:
    - `scripts/local_e2e_matrix.ps1`
      - now writes:
        - `requested_primary_suppliers`
        - `requested_english_fallback_suppliers`
        - `requested_isolation_supplier`
        - `expected_route_key`
    - `scripts/local_residue_audit.ps1`
      - now falls back to `expected_route_key` and
        `requested_isolation_supplier` when runtime evidence is absent
  - live verification:
    - `20260620-113450-327-e2e-matrix`
      - requested supplier `xunlei`
      - `job_terminal_status=2`
      - `job_error_info=No Sub Found`
      - `route_key=movie.safe_fail`
      - `checks.no_sub_safe_failure=passed`
    - `20260620-113548-319-e2e-matrix`
      - requested supplier `shooter`
      - `job_terminal_status=2`
      - `job_error_info=No Sub Found`
      - `route_key=movie.safe_fail`
      - `checks.no_sub_safe_failure=passed`
  - residue verification after the fix:
    - the latest safe-fail route proof was kept
    - the extra isolated `xunlei` safe-fail proof was also kept for the same
      route as distinct supplier evidence
  - cleanup execution was then run immediately and verified:
    - stale duplicate report directories were removed
    - active candidate image/container and sample pools remained intact

- the same summary-retention repair is now also proven on the series side:
  - `20260620-114153-260-e2e-matrix`
    - sample:
      - `黑袍纠察队 - S01E02 - 第 2 集`
    - requested supplier:
      - `xunlei`
    - result:
      - `job_terminal_status=2`
      - `job_error_info=No Sub Found`
      - `requested_isolation_supplier=xunlei`
      - `expected_route_key=series.safe_fail`
      - `route_key=series.safe_fail`
      - `checks.no_sub_safe_failure=passed`
- route coverage was refreshed after that round:
  - `route-coverage-snapshot-20260620-114233-192.json`
  - `coverage_ok=true`
  - `missing_required_route_count=0`
- supplier status was also refreshed after cleanup:
  - `supplier-status-20260620-114124-315`
  - all prior role conclusions held
  - `subtitle_best` remained the only disabled / unverified supplier role

- content-audit instrumentation was then improved so the retained accuracy view
  can distinguish genuine bilingual presentation from suspicious English
  leakage:
  - `scripts/local_subtitle_content_audit.ps1`
    - now captures:
      - `english_only_samples`
      - `mixed_language_samples`
      - `bilingual_presentation_samples`
      - `looks_bilingual_presentation`
  - refreshed audit:
    - `subtitle-content-audit-20260620-114959-484.json`
  - key conclusion:
    - residual issues are now concentrated in proper nouns, acronyms, and a few
      OCR-style single-letter fragments
    - there is no longer evidence of large untranslated English dialogue blocks
      leaking through the retained translated routes

- one more real runtime LLM recheck was then completed after tightening the
  prompt instructions for recurring named entities:
  - local verification passed:
    - `py_compile`
    - `subflow.test_translate_job`
    - `subflow.test_openai_compatible_client`
    - `go test ./pkg/logic/pre_download_process ./pkg/downloader -count=1`
  - runtime round:
    - `20260620-115641-939-e2e-matrix`
    - sample:
      - `黑袍纠察队 - S01E03 - 第 3 集`
    - route:
      - `series.llm_fallback`
    - result:
      - `job_terminal_status=3`
      - `route_assertion=passed`
      - `llm_output_language=passed`
  - refreshed content audit showed a real reduction in pure-English residue on
    this sample:
    - earlier retained proof:
      - `8 / 1293`
    - new retained proof:
      - `2 / 1286`
  - remaining pure-English residues are now down to:
    - `Compound`
    - `M.M.`

- the same tightened prompt was also re-checked on the retained movie sample:
  - `20260620-123803-097-e2e-matrix`
  - sample:
    - `星际穿越 (2014) - 2160p`
  - route:
    - `movie.llm_fallback`
  - result:
    - `job_terminal_status=3`
    - `route_assertion=passed`
    - `llm_output_language=passed`
  - refreshed content audit showed another small but real reduction in
    pure-English residue:
    - earlier retained movie proof:
      - `7 / 2942`
    - new retained movie proof:
      - `6 / 2939`
  - latest remaining pure-English movie residues are now almost entirely:
    - `NASA`
    - `TARS`
    - `CASE`
  - runtime-image drift was then proved and corrected before keeping newer
    LLM evidence:
    - the workspace `translate_job.py` already had the newest
      named-entity/postprocess logic
    - the old running candidate container did not
    - rebuild evidence:
      - `D:\tmp\csf-local-candidate\reports\20260620-141756-453`
    - new active image:
      - `sha256:8535f02585a6cb51bb42b7edc7e5abc031a7ac7e1cb2581403b8f90ead270d0b`
    - post-rebuild movie rerun:
      - `20260620-142303-552-e2e-matrix`
      - `english_only_line_count=3 / dialogue_line_count=2947`
    - post-rebuild series rerun:
      - `20260620-144314-014-e2e-matrix`
      - `english_only_line_count=0 / dialogue_line_count=1283`
  - conclusion:
    - movie-side residuals are now firmly in the “policy choice” zone rather
      than the “translation failed” zone
Addendum on 2026-06-20 after full non-LLM recheck on the same active rebuilt image:

- the active image/container pairing was first re-confirmed:
  - image:
    - `sha256:8535f02585a6cb51bb42b7edc7e5abc031a7ac7e1cb2581403b8f90ead270d0b`
  - container:
    - `chinesesubfinder-local-candidate`
    - running that exact image
- non-LLM routes were then re-run on that same active image without another
  rebuild:
  - `20260620-150459-388-e2e-matrix`
    - `movie.native_chinese`
    - `actual_supplier=subhd`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - `20260620-150545-259-e2e-matrix`
    - `movie.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `route_assertion=passed`
  - `20260620-150733-907-e2e-matrix`
    - `movie.english_fallback`
    - `actual_supplier=subdl`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - `20260620-150805-106-e2e-matrix`
    - `movie.safe_fail`
    - `job_error_info=No Sub Found`
    - `checks.no_sub_safe_failure=passed`
  - `20260620-150834-373-e2e-matrix`
    - `series.native_chinese`
    - `actual_supplier=subhd`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - `20260620-150936-408-e2e-matrix`
    - `series.english_fallback`
    - `actual_supplier=subdl`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - `20260620-151007-088-e2e-matrix`
    - `series.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `route_assertion=passed`
- refreshed active-image snapshots also landed:
  - route coverage:
    - `route-coverage-snapshot-20260620-151321-892.json`
    - `missing_required_route_count=0`
  - content audit:
    - `subtitle-content-audit-20260620-151329-020.json`
  - supplier status:
    - `supplier-status-20260620-151322-817`
    - valid:
      - `xunlei`
      - `shooter`
      - `assrt`
      - `subdl`
      - `opensubtitles`
      - `tvsubtitles`
      - `moviesubtitles`
      - `subtitlecat`
      - `subhd`
    - invalid:
      - `subtitle_best`
- current conclusion after this pass:
  - the active rebuilt image now has fresh proof for:
    - all required non-LLM routes
    - both LLM fallback routes
  - the remaining incomplete point is still only `subtitle_best`

Addendum on 2026-06-20 after subtitle_best boundary tests:

- two narrow regression tests were added so the remaining `subtitle_best`
  boundary is not only documented but enforced:
  - `pkg/logic/pre_download_process/pre_download_proces_test.go`
    - `subtitle_best` supplier role must stay out of collected supplier plans
      when the API key is empty
    - `subhd` must still remain present in the primary Chinese chain in that
      same scenario
  - `pkg/logic/sub_supplier/status_probe_test.go`
    - `subtitle_best` must report `credential missing` when enabled without a
      key
- focused verification passed:
  - `go test ./pkg/logic/pre_download_process -count=1`
  - `go test ./pkg/logic/sub_supplier -count=1`
  - `go test ./pkg/subtitle_best_api -count=1`
- this still does not convert `subtitle_best` into a runtime-proved supplier,
  but it does remove more drift risk from the current intended boundary

Addendum on 2026-06-20 after subtitle_best shared-code log-noise downgrade:

- one more low-risk runtime polish change was applied:
  - when `subtitle_best` shared code is unavailable because the auth key is not
    configured, `PreDownloadProcess.Init()` now logs that condition at `INFO`
    instead of `WARNING`
- focused verification passed:
  - `go test ./pkg/logic/pre_download_process -count=1`
  - `go test ./pkg/logic/sub_supplier -count=1`
- rebuilt runtime proof:
  - active image is now:
    - `sha256:638cb652520a5d6c70864a840b1ab1998b8c47ae3139d3aae90cee3866f32dbb`
  - active container log shows:
    - `[INFO] ... SubtitleBestCodeProvider.GetCode auth key is not set continue without shared code`
  - the earlier misleading `[WARNING]` form is no longer emitted by the rebuilt
    startup path
- this does not change download-chain behavior, but it does reduce false
  operational noise in the current intended disabled-`subtitle_best` runtime

Addendum on 2026-06-20 after subtitle_best default-order de-drift:

- one last code-level inconsistency was removed from the default chain policy:
  - before this change, `DefaultPrimarySubSiteSequence()` still placed
    `subtitle_best` at the front even though:
    - the pulled FnOS config keeps it disabled
    - supplier-plan collection only includes it when `enabled && api_key != ""`
    - the verified current policy already treats the default native-Chinese
      stage as `assrt -> subhd -> shooter -> xunlei -> opensubtitles`
- the constants and tests were aligned to the intended current policy:
  - `pkg/types/common/sub_site_sequence.go`
  - `pkg/types/common/sub_site_sequence_test.go`
  - `pkg/logic/pre_download_process/pre_download_proces_test.go`
- focused verification passed:
  - `go test ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier -count=1`
- conclusion after this addendum:
  - this does not add live supplier proof for `subtitle_best`
  - it does remove a real future drift risk by ensuring the default ordering
    constants no longer claim `subtitle_best` is a first-tier default supplier

Addendum on 2026-06-20 after rebuilt-image smoke verification:

- the local candidate runtime was rebuilt once more after the default-order
  de-drift so the active container would no longer lag behind the current
  workspace:
  - active image now:
    - `sha256:712d52ca807be105c511c7322411d50dd01dae006088bc687e9ad81cb396ea15`
  - active container stayed:
    - `chinesesubfinder-local-candidate`
- one narrow smoke round was run immediately after the rebuild against the real
  pulled FnOS config volumes:
  - e2e proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-154252-798-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese disabled
    - English fallback isolated to `subdl`
  - result:
    - `route_key=movie.english_fallback`
    - `actual_supplier=subdl`
    - `job_terminal_status=3`
    - `policy_warnings=[]`
- conclusion after this addendum:
  - the active local runtime is now aligned with the current workspace again
  - the rebuild did not regress the verified English fallback path

Addendum on 2026-06-20 after additional current-image route proofs:

- two more narrow live checks were added on the same active `712d52ca...`
  image so the current runtime is not represented by only one English-fallback
  smoke round:
  - native Chinese proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-154810-112-e2e-matrix\e2e-summary.json`
    - `route_key=series.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
  - explicit translated fallback proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-154930-032-e2e-matrix\e2e-summary.json`
    - `route_key=series.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `job_terminal_status=3`
- conclusion after this addendum:
  - the current active image now has fresh live proofs across three route
    stages:
    - English fallback
    - native Chinese primary chain
    - explicit translated-Chinese fallback
  - the remaining incomplete parts are no longer about whether the current
    rebuilt runtime works at all; they are about how much of the full retained
    evidence window should be refreshed on this exact image versus trusted from
    the immediately preceding image

Addendum on 2026-06-20 after movie-branch current-image checks:

- two more direct movie-branch rounds were added on the same active
  `712d52ca...` image:
  - movie native-Chinese proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-155530-524-e2e-matrix\e2e-summary.json`
    - `route_key=movie.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
  - movie safe-fail proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-155634-014-e2e-matrix\e2e-summary.json`
    - `route_key=movie.safe_fail`
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
- conclusion after this addendum:
  - the current active image now has refreshed proofs for:
    - movie.native_chinese
    - movie.english_fallback
    - movie.safe_fail
    - series.native_chinese
    - series.subtitlecat_translated
  - the most valuable still-unrefreshed proof on this exact image is now the
    LLM fallback path, not the core non-LLM movie/series route skeleton

Addendum on 2026-06-20 after LLM repair-prompt tightening:

- the previous movie LLM proof still showed a tiny but real leftover class:
  short plain-English dialogue fragments such as `- In secret.` and
  `- Why secret?`
- a small prompt-only refinement was added in:
  - `third_party/subflow/src/subflow/translate_job.py`
  - with a matching unit test in:
    - `third_party/subflow/src/subflow/test_translate_job.py`
- focused verification passed:
  - `python -m unittest subflow.test_translate_job`
- the candidate image was then rebuilt again and re-verified on the same movie
  LLM path:
  - active image now:
    - `sha256:3d7f32f87354d30895845e737afa307ddca4f5850ba27b33c035e48b1c939dab`
  - rebuilt-image proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-164047-399-e2e-matrix\e2e-summary.json`
    - `route_key=movie.llm_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
- rebuilt-image content audit outcome:
  - `english_only_line_count` dropped to `4`
  - the remaining English-only samples are now only:
    - `- NASA？`
    - `- NASA。`
    - `1G。`
    - `RPM。`
  - the previously visible missed short-dialogue English lines are no longer in
    the retained sample set
- conclusion after this addendum:
  - the newest active image is aligned with the workspace again
  - the LLM path on that newest image is now materially cleaner than the prior
    movie rerun, and the remaining English-only residues are within the allowed
    acronym / technical-token envelope

Addendum on 2026-06-20 after same-image rerun on `dc20fe0945ce...`:

- the LLM naturalization pass was tightened one more time in local code:
  - `_cue_kind()` no longer treats short all-caps cues with trailing dialogue
    punctuation such as `NASA?` as screen text
  - `_is_allowed_english_only_line()` no longer whitelists punctuated acronym
    lines as acceptable raw-English dialogue output
  - both the base translate prompt and repair prompt now explicitly require
    natural Chinese rendering for isolated acronym/alphanumeric shorthand cues
    such as `NASA?`, `RPM.`, `1G.`, and `2G` when they function as spoken
    dialogue
- focused verification passed locally:
  - `python -m unittest subflow.test_translate_job`
  - `go test ./pkg/downloader ./pkg/logic/mark_system ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier -count=1`
- the local candidate image was rebuilt again and is now:
  - `sha256:dc20fe0945ce...`
- same-image route proofs completed on this image:
  - `movie.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-173602-799-e2e-matrix\e2e-summary.json`
    - `actual_supplier=subhd`
  - `movie.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-173709-104-e2e-matrix\e2e-summary.json`
    - `actual_supplier=subtitlecat_translated`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-180330-023-e2e-matrix\e2e-summary.json`
    - `actual_supplier=subhd`
  - `series.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-180702-159-e2e-matrix\e2e-summary.json`
    - `actual_supplier=subtitlecat_translated`
  - `movie.safe_fail`
    - `D:\tmp\csf-local-candidate\reports\20260620-181014-844-e2e-matrix\e2e-summary.json`
    - `job_error_info=No Sub Found`
- same-image movie LLM proof also completed:
  - `D:\tmp\csf-local-candidate\reports\20260620-174022-018-e2e-matrix\e2e-summary.json`
  - `route_key=movie.llm_fallback`
  - `actual_supplier=subtitlecat`
  - `job_terminal_status=3`
- latest movie LLM content audit on the active image now shows:
  - `dialogue_line_count=2940`
  - `english_only_line_count=0`
  - remaining mixed-language tokens are only embedded accepted items such as
    `NASA`, `GPS`, `MRI`, `A计划`, and `B计划`
- one same-image route remains blocked by pulled-config reality rather than code
  state:
  - the attempted `movie.english_fallback` step in `scripts/local_full_acceptance.ps1`
    failed immediately with:
    `Requested supplier prerequisites are missing in the pulled FnOS config: subdl api key`
  - therefore the default `subdl`-led English fallback chain still cannot be
    marked re-proved on the newest image without first obtaining a config that
    actually includes the `subdl` credential
- another narrow harness-quality issue was surfaced:
  - the newest `series.native_chinese` and `series.subtitlecat_translated`
    `e2e-summary.json` files carry malformed JSON escaping in sample-name
    fields when the source sample path contains damaged Chinese characters
  - route key, winning supplier, terminal status, and final-output fields still
    exist in the files, so current acceptance evidence remains usable

Addendum on 2026-06-20 after latest-image movie English-fallback completion:

- one more latest-image acceptance round was run on the same active image
  `dc20fe0945ce...` for the missing route category:
  - `D:\tmp\csf-local-candidate\reports\20260620-181822-050-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese suppliers: `__none__`
    - English fallback suppliers: `subtitlecat`
  - result:
    - `route_key=movie.english_fallback`
    - `route_stage=english_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
    - `final_output_has_chinese=false`
- this completes the objective's required latest-image route coverage set:
  - `movie.native_chinese`
  - `series.native_chinese`
  - `movie.english_fallback`
  - `series.subtitlecat_translated`
  - `safe_fail`
- the remaining nuance is narrower than the goal requirement:
  - the pulled FnOS config still lacks a `subdl` key, so the exact default
    `subdl`-first English fallback order cannot currently be re-proved on this
    image
  - however the latest image now does contain a valid `movie.english_fallback`
    success proof through the currently available pulled-config supplier
    `subtitlecat`

Addendum on 2026-06-20 after subtitle_best mechanism audit completion:

- `subtitle_best` was re-confirmed as two separate mechanisms sharing one name:
  - direct supplier role:
    - `pkg/logic/sub_supplier/subtitle_best/subtitle_best.go`
    - `pkg/logic/sub_supplier/subtitle_best/api.go`
    - uses `https://api.subtitle.best/share-sub/v1/*`
    - requires `subtitle_best_settings.enabled=true` and
      `subtitle_best_settings.api_key` non-empty
  - shared support-service role:
    - `pkg/subtitle_best_api/subtitle_best_api.go`
    - `pkg/media_info_dealers/dealers.go`
    - `pkg/logic/pre_download_process/pre_download_proces.go`
    - uses `https://api.subtitle.best/v1/subhd-code`,
      `/v1/media-info`, `/v1/id-convert`, `/v1/feedback`
    - requires a valid startup `CustomAuth` triple, not the supplier
      `subtitle_best_settings.api_key`
- current failure mode is therefore split, not singular:
  - supplier role is inactive because the pulled config keeps
    `subtitle_best_settings.enabled=false` and `api_key=""`
  - shared support APIs are inactive whenever runtime `CustomAuth` is absent or
    still on the placeholder defaults
- minimum revival paths:
  - revive supplier downloads only:
    - provide real `subtitle_best api_key`
    - enable the supplier in settings
    - run isolated movie / series runtime probes
  - revive shared support only:
    - provide real `CustomAuth`
    - verify `subhd-code`, media-info fallback, and id-convert
  - fully revive subtitle.best behavior:
    - provide both the real supplier API key and real `CustomAuth`
