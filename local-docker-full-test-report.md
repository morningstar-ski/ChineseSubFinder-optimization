# Local Docker Full Test Report

Last updated: 2026-06-20 17:02 Asia/Shanghai

## Scope

This report records the current verified local-first acceptance state for
`ChineseSubFinder-provider-pack` against the real mounted library at
`\\192.168.100.4\video\link`.

Validation stayed inside:

- image: `chinesesubfinder:local-candidate`
- container: `chinesesubfinder-local-candidate`
- local root: `D:\tmp\csf-local-candidate`
- sample spec pool: `D:\tmp\csf-real-media-stage`

## Current Verified Evidence

### Latest provider-source verification under pulled FnOS config

- full FnOS config and browser state were re-pulled again before this proof:
  - `D:\tmp\fnos-csf-config-pull-20260619-194339.json`
  - verified:
    - `remote_settings_sha256=9e8dd16daca715bce84f59778a6471c067d5ee007b4b704ee209b198f981f92b`
    - `volume_settings_sha256=9e8dd16daca715bce84f59778a6471c067d5ee007b4b704ee209b198f981f92b`
- candidate root stayed on the local Docker runtime while reusing the pulled
  FnOS CSF settings snapshot
- after the repull, the local candidate was restarted through
  `scripts/local_candidate_round.ps1` so the working volume kept the pulled
  supplier settings while remapping runtime paths back to `/media/movies` and
  `/media/series`
- timeline-fix boundary was tightened so long media without usable embedded
  subtitles no longer block job completion on full-audio VAD extraction
- current provider-focused movie proofs on the real mounted library:
  - `assrt` series native Chinese under the re-pulled full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-200709-664-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=series.native_chinese`,
      `final_output_has_chinese=true`
    - container log proof:
      - `OrgSubName: [assrt]_0_【NEW自译】黑袍纠察队...S01E03...ass`
    - cleanup result:
      - generated `.csf-bk` residue was removed from
        `\\192.168.100.4\video\link\电视剧\欧美剧\黑袍纠察队 (2019)\Season 1`
  - `subhd` series native Chinese under the re-pulled full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-201239-517-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=series.native_chinese`,
      `final_output_has_chinese=true`
    - container log proof:
      - `subhd download gate passed without captcha`
      - `OrgSubName: [subhd]_0_03.ass`
    - cleanup result:
      - generated `.csf-bk` residue was removed from
        `\\192.168.100.4\video\link\电视剧\欧美剧\黑袍纠察队 (2019)\Season 1`
  - `subdl` isolated English fallback under the re-pulled full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-201123-934-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=2`, `job_error_info=No Sub Found`,
      `route_key=series.safe_fail`
    - container log proof:
      - `subdl ... Try Search Query map[api_key:... imdb_id:tt1190634 ...]`
      - `Queue 1 subdl Covered all needed episodes in metadata`
      - `all site download sub not found`
  - `moviesubtitles` isolated English fallback regression under the re-pulled
    full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-201535-498-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=movie.english_fallback`,
      `final_output_has_chinese=false`
    - container log proof:
      - `OrgSubName: [moviesubtitles]_0_Silence.720p.BlueRay.BLOW.en.srt`
    - cleanup result:
      - no subtitle residue remained in
        `\\192.168.100.4\video\link\电影\外语电影\沉默的羔羊 (1991)` after the round
  - `subtitlecat_translated` isolated explicit Chinese fallback under the
    re-pulled full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-202802-077-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`,
      `route_key=series.subtitlecat_translated`,
      `final_output_has_chinese=true`
    - container log proof:
      - `Queue 1 subtitlecat_translated Start...`
      - `OrgSubName: [subtitlecat_translated]_0_洛佩兹一家 _S1E1.srt`
    - cleanup result:
      - generated `.csf-bk` residue was removed from
        `\\192.168.100.4\video\link\电视剧\欧美剧\洛佩兹一家 (2002)\Season 1`
  - `subtitlecat` default English fallback under the re-pulled full FnOS
    config:
    - `D:\tmp\csf-local-candidate\reports\20260619-203110-654-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=series.english_fallback`,
      `final_output_has_chinese=false`
    - container log proof:
      - `Queue 1 subtitlecat Start...`
      - `OrgSubName: [subtitlecat]_0_洛佩兹一家 _S1E1.srt`
    - cleanup result:
      - generated `.csf-bk` residue was removed from
        `\\192.168.100.4\video\link\电视剧\欧美剧\洛佩兹一家 (2002)\Season 1`
  - `opensubtitles` English fallback under the re-pulled full FnOS config:
    - `D:\tmp\csf-local-candidate\reports\20260619-195837-520-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=movie.english_fallback`,
      `final_output_has_chinese=false`
    - container log proof:
      - `OrgSubName: [opensubtitles]_0_记忆碎片_S0E1080.srt`
    - cleanup result:
      - generated sample subtitle residue was removed from
        `\\192.168.100.4\video\link\电影\外语电影\记忆碎片 (2000)` after evidence capture
  - `assrt`
    - `D:\tmp\csf-local-candidate\reports\20260619-140132-637-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `final_output_has_chinese=true`
  - `opensubtitles`
    - `D:\tmp\csf-local-candidate\reports\20260619-140458-527-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `final_output_has_chinese=true`
  - `subdl`
    - `D:\tmp\csf-local-candidate\reports\20260619-140303-575-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, but current sampled output stayed
      English on this title and still needs follow-up analysis

### Core routes

- `movie.native_chinese`
  - `D:\tmp\csf-local-candidate\reports\20260619-072406-849-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-054245-947-e2e-matrix\e2e-summary.json`
- `movie.subtitlecat_translated`
  - `D:\tmp\csf-local-candidate\reports\20260619-072444-724-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-054334-019-e2e-matrix\e2e-summary.json`
- `movie.english_fallback`
  - `D:\tmp\csf-local-candidate\reports\20260619-075032-805-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-074147-524-e2e-matrix\e2e-summary.json`
- `movie.safe_fail`
  - `D:\tmp\csf-local-candidate\reports\20260619-072626-440-e2e-matrix\e2e-summary.json`
- `series.native_chinese`
  - `D:\tmp\csf-local-candidate\reports\20260619-075153-872-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-074307-305-e2e-matrix\e2e-summary.json`
- `series.english_fallback`
  - `D:\tmp\csf-local-candidate\reports\20260619-075055-866-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-074210-420-e2e-matrix\e2e-summary.json`
- `series.subtitlecat_translated`
  - `D:\tmp\csf-local-candidate\reports\20260619-072827-356-e2e-matrix\e2e-summary.json`
  - `D:\tmp\csf-local-candidate\reports\20260619-054727-650-e2e-matrix\e2e-summary.json`
- `movie.llm_fallback`
  - `D:\tmp\csf-local-candidate\reports\20260619-084650-122-e2e-matrix\e2e-summary.json`
- `series.llm_fallback`
  - `D:\tmp\csf-local-candidate\reports\20260619-090711-552-e2e-matrix\e2e-summary.json`

### Route coverage snapshot

- retained snapshot:
  - `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260619-181953-522.json`
- current result:
  - `present_route_count=10`
  - `missing_required_route_count=0`
  - `coverage_ok=true`
  - optional `series.safe_fail` is also now present in the retained evidence

### Spot checks

- native movie Chinese output remained readable through the live `subhd`
  captcha gate:
  - `D:\tmp\csf-local-candidate\reports\20260619-072406-849-e2e-matrix`
- native series bilingual ASS remained aligned and readable:
  - `D:\tmp\csf-local-candidate\reports\20260619-075153-872-e2e-matrix`
- English fallback SRT remained clean plain text:
  - `D:\tmp\csf-local-candidate\reports\20260619-075055-866-e2e-matrix`
- explicit translated-Chinese route produced readable Chinese dialogue:
  - `D:\tmp\csf-local-candidate\reports\20260619-072827-356-e2e-matrix`
- latest full-movie LLM fallback stayed in Chinese:
  - `D:\tmp\csf-local-candidate\reports\20260619-084650-122-e2e-matrix`
- latest series LLM fallback stayed in Chinese:
  - `D:\tmp\csf-local-candidate\reports\20260619-090711-552-e2e-matrix`

## Script Structure

Shared acceptance definitions now live in:

- `scripts/local_acceptance_matrix.psd1`
- `scripts/local_acceptance_runner.ps1`

Batch entrypoints now stay thin:

- `scripts/local_full_acceptance.ps1`
- `scripts/local_expanded_acceptance.ps1`
- `scripts/local_llm_acceptance.ps1`

The route coverage and cleanup proof chain is:

- `scripts/local_route_coverage_snapshot.ps1`
- `scripts/local_residue_audit.ps1`
- `scripts/local_cleanup.ps1`

## Current Chain Policy

The local candidate now follows a clearer three-stage structure:

- native Chinese stage:
  - `assrt`
  - `subhd`
  - `shooter`
  - `xunlei`
  - `opensubtitles`
- English fallback stage:
  - `opensubtitles`
  - `subdl`
  - `subtitlecat`
  - `moviesubtitles`
- translated fallback stage:
  - `subtitlecat_translated` only when explicitly enabled
  - LLM fallback only when explicitly enabled

The key policy change in this round is:

- `subdl`, `tvsubtitles`, and `moviesubtitles` no longer participate in the
  native-Chinese primary chain by default
- `subdl` and `moviesubtitles` remain available in the default English fallback
  layer
- `tvsubtitles` remains implemented and probeable, but not wired into the
  active default English fallback chain
- the default provider order was also tightened so `opensubtitles` is ahead of
  `subdl` in the shared site sequence

Update on 2026-06-20 after additional isolated real-config reruns:

- `subdl` now has fresh isolated supplier proofs in both layers that matter for
  the current policy:
  - series English fallback:
    `D:\tmp\csf-local-candidate\reports\20260620-004142-371-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_key=series.english_fallback`
  - movie English fallback:
    `D:\tmp\csf-local-candidate\reports\20260620-004512-354-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_key=movie.english_fallback`
- `tvsubtitles` also got a true isolated rerun with the primary Chinese stage
  explicitly disabled:
  - `D:\tmp\csf-local-candidate\reports\20260620-004014-640-e2e-matrix`
  - result stayed `job_error_info=No Sub Found`
  - this no longer just means "untested"; it means the current sampled evidence
    does not justify keeping `tvsubtitles` in the default English fallback
    chain
- the runtime policy was tightened again after those reruns:
  - `subdl` stays English-fallback-only
  - `tvsubtitles` is no longer registered in the default download chain
  - `tvsubtitles` remains implemented and probeable, but it is no longer part
    of the default fallback posture
- `subtitle_best` was traced through code as both:
  - a direct subtitle supplier backed by `https://api.subtitle.best/share-sub/v1`
  - an auxiliary shared service used for `subhd` shared-code fetches and
    metadata / ID-conversion fallback through `pkg/subtitle_best_api`
  - the pulled FnOS config still has `subtitle_best_settings.api_key=""`, so
    the supplier role remains unverified in live runtime

Update on 2026-06-20 after the rebuilt-image staged-library reruns:

- `subdl` still works on the current candidate image after the latest route
  policy change:
  - `D:\tmp\csf-local-candidate\reports\20260620-010357-745-e2e-matrix`
  - proves:
    - `actual_supplier=subdl`
    - `route_key=series.english_fallback`
    - `final_output_has_chinese=false`
- `tvsubtitles` still fails under true isolation on the same rebuilt image:
  - `D:\tmp\csf-local-candidate\reports\20260620-010702-382-e2e-matrix`
  - proves:
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
    - `route_key=series.safe_fail`
- `subtitlecat` default English fallback still works on the rebuilt image:
  - `D:\tmp\csf-local-candidate\reports\20260620-010755-329-e2e-matrix`
  - proves:
    - `actual_supplier=subtitlecat`
    - `route_key=series.english_fallback`
    - `final_output_has_chinese=false`
- `subtitlecat_translated` explicit Chinese fallback also still works on the
  rebuilt image:
  - `D:\tmp\csf-local-candidate\reports\20260620-010849-101-e2e-matrix`
  - proves:
    - `actual_supplier=subtitlecat_translated`
    - `route_key=series.subtitlecat_translated`
    - `final_output_has_chinese=true`
- staged media roots were checked after each round and did not retain generated
  subtitle residue; only the copied report artifacts under
  `D:\tmp\csf-local-candidate\reports\...` remain as evidence

Update on 2026-06-20 after direct-mount and runtime-pool hardening:

- the previous sample-pool audit was too weak: it only proved host-side file
  existence and did not prove Docker visibility
- the audit script now also checks whether each sample is visible through an
  actual Docker bind mount
  - direct real-library sample pool audit:
    `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260620-012836-871.json`
    - `existing=10`
    - `docker_visible=0`
    - current Docker bind source derived from the UNC roots is
      `/run/desktop/mnt/host/uC/192.168.100.4/video/link/...`
    - even with the corrected mount bridge, all 10 intended samples were still
      invisible inside Docker on this machine
- because of that host-specific Docker bridge failure, a new local helper now
  materializes a minimized runtime sample pool from the same FnOS library:
  - script:
    `scripts/local_materialize_sample_pool.ps1`
  - output root:
    `D:\tmp\csf-real-media-runtime`
  - materialize report:
    `D:\tmp\csf-real-media-runtime\materialize-report.json`
  - copied files: `30`
- the materialized pool was then re-audited and passed both host and Docker
  visibility checks:
  - `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260620-020123-986.json`
  - `existing=10`
  - `docker_visible=10`
- the acceptance wrappers were also tightened so they can reuse the pulled FnOS
  config/browser Docker volumes instead of silently falling back to local empty
  config roots:
  - `scripts/local_acceptance_runner.ps1`
  - `scripts/local_expanded_acceptance.ps1`
  - `scripts/local_full_acceptance.ps1`
  - `scripts/local_llm_acceptance.ps1`
- the expanded non-LLM acceptance profile now restarts the candidate container
  on its first round so the active sample pool cannot drift from the mounted
  media root
- with the materialized runtime pool and FnOS working volumes, the expanded
  batch re-passed on 2026-06-20:
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-020331-265-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_assertion=passed`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-020423-943-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_assertion=passed`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-020459-622-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_assertion=passed`
- residue policy was updated so `D:\tmp\csf-real-media-runtime` is treated as an
  active sample pool instead of a stale temp artifact, and a cleanup pass was
  executed after verification to drop invalid or superseded report directories
  while keeping the current evidence window

Update on 2026-06-20 after failure-path restore hardening:

- `scripts/local_e2e_matrix.ps1` now serializes shared-container route rounds
  with a mutex keyed by `CandidateContainer + BaseUrl + ConfigDockerVolume`
- the same script now restores the pulled FnOS working config before rethrowing
  a failed round instead of calling `Write-Error` first and aborting cleanup
- `scripts/local_candidate_round.ps1` now keeps a stable local baseline copy at:
  - `D:\tmp\csf-local-candidate\artifacts\baseline-settings.json`
- fresh failure-path proof:
  - `D:\tmp\csf-local-candidate\reports\20260620-040604-670-e2e-matrix`
  - result:
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
- fresh post-failure runtime/provider-state proof:
  - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-040650-854\summary.json`
  - result:
    - `assrt`, `subdl`, `opensubtitles`, `tvsubtitles`, `moviesubtitles`,
      `subtitlecat`, and `subhd` all returned to `enabled=true` / `valid=true`
    - `subtitle_best` remains `enabled=false` / `valid=false`

Update on 2026-06-20 after runtime-pool auto-resolution and full-chain refresh:

- the acceptance runner now auto-resolves the requested sample pool:
  - direct Docker-visible pool -> use as-is
  - Docker-invisible real-library pool -> reuse or rebuild
    `D:\tmp\csf-real-media-runtime\sample-specs`
- this behavior was verified by running the wrappers again against the original
  stage root `D:\tmp\csf-real-media-stage`, which printed:
  - `Reuse materialized runtime sample pool: D:\tmp\csf-real-media-runtime\sample-specs`
- LLM-only acceptance then re-passed on the auto-resolved runtime pool with the
  pulled FnOS working volumes:
  - `movie.llm_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-022515-500-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_assertion=passed`
    - output sample is simplified Chinese
  - `series.llm_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-024514-233-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_assertion=passed`
    - output sample is simplified Chinese
- a full non-LLM acceptance refresh also re-passed on the same auto-resolved
  runtime pool, giving current proofs for the remaining default-chain routes:
  - `movie.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-025717-754-e2e-matrix`
    - `actual_supplier=subhd`
  - `movie.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-025816-538-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-030020-050-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `movie.safe_fail`
    - `D:\tmp\csf-local-candidate\reports\20260620-030058-248-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-030132-231-e2e-matrix`
    - `actual_supplier=subhd`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-030243-315-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-030453-832-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
- after the full refresh, route coverage again remained complete for all 9
  required routes

## Verification Commands

```powershell
go test ./internal/backend/controllers/v1 ./pkg/logic/sub_supplier/subtitlecat -count=1
go test ./pkg/logic/sub_supplier/subhd -count=1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\local_full_acceptance.ps1 `
  -WorkspaceRoot 'C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack' `
  -CandidateRoot 'D:\tmp\csf-local-candidate' `
  -CandidateImage 'chinesesubfinder:local-candidate' `
  -CandidateContainer 'chinesesubfinder-local-candidate' `
  -LLMProvider 'deepseek' `
  -LLMBaseUrl 'https://api.deepseek.com' `
  -LLMModel 'deepseek-v4-flash' `
  -LLMApiKey '<session key>'
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\local_expanded_acceptance.ps1 `
  -WorkspaceRoot 'C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack' `
  -CandidateRoot 'D:\tmp\csf-local-candidate' `
  -CandidateImage 'chinesesubfinder:local-candidate' `
  -CandidateContainer 'chinesesubfinder-local-candidate'
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\local_llm_acceptance.ps1 `
  -WorkspaceRoot 'C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack' `
  -CandidateRoot 'D:\tmp\csf-local-candidate' `
  -CandidateImage 'chinesesubfinder:local-candidate' `
  -CandidateContainer 'chinesesubfinder-local-candidate' `
  -LLMProvider 'deepseek' `
  -LLMBaseUrl 'https://api.deepseek.com' `
  -LLMModel 'deepseek-v4-flash' `
  -LLMApiKey '<session key>'
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\local_sample_pool_audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\local_route_coverage_snapshot.ps1
```

## Operational Notes

- The full matrix remains the authoritative end-to-end driver for the complete
  chain.
- Real FnOS library validation must start the candidate container with the same
  sample-spec-derived `DockerMoviesSource` and `DockerSeriesSource` that the
  e2e round will use; otherwise the container keeps the default local sample
  mount and the queued real-media path is deleted as missing.
- `scripts/local_e2e_matrix.ps1` now fails fast when the requested
  `container_video_path` is not actually mounted inside
  `chinesesubfinder-local-candidate`, instead of letting the job time out in
  status `0`.
- Pulled FnOS runtime state is now split into two layers locally:
  - immutable baseline volumes:
    - `csf_fnos_config_full_20260619`
    - `csf_fnos_browser_full_20260619`
  - refreshable clean working volumes for actual test rounds:
    - `csf_fnos_config_working`
    - `csf_fnos_browser_working`
- The refresh helper is:
  - `scripts/refresh_fnos_working_volumes.ps1`
- The full FnOS config repull helper is:
  - `scripts/pull_fnos_full_config.ps1`
- The current live FnOS CSF config source for local reuse is:
  - `fnos-csf:/vol1/1000/docker/csf/config`
  - `fnos-csf:/vol1/1000/docker/csf/browser`
- The working config reset clears stale queue/cache state while preserving the
  pulled settings and browser runtime, so local tests keep the real supplier
  credentials without replaying old FnOS jobs.
- The latest repull manifest is:
  - `D:\tmp\fnos-csf-config-pull-20260619-194339.json`
- The latest repull verified:
  - `remote_settings_sha256=9e8dd16daca715bce84f59778a6471c067d5ee007b4b704ee209b198f981f92b`
  - `volume_settings_sha256=9e8dd16daca715bce84f59778a6471c067d5ee007b4b704ee209b198f981f92b`
- The matrix harness no longer replays a redacted `GET /v1/settings` payload
  back into `PUT /v1/settings`; runtime route changes are now derived from the
  local full-config shadow copy, so provider tokens and credentials are not
  silently wiped during tests.
- The explicit `subtitlecat_translated` route harness now isolates that path by
  default when no supplier sets are explicitly provided:
  - `scripts/local_e2e_matrix.ps1`
  - this prevents `assrt` or `subhd` from winning the round and then being
    misclassified as `subtitlecat_translated`
- The matrix harness now also records actual winning supplier evidence from
  container logs:
  - `scripts/local_e2e_matrix.ps1`
  - per-round evidence is written to `supplier-evidence.json`
  - single-supplier isolation rounds now assert both `route_key` and
    `actual_supplier`
- Route isolation no longer clears supplier secrets just because a supplier is
  disabled for the current round:
  - `scripts/local_e2e_matrix.ps1`
  - the harness now relies on `enabled=false` plus supplier daily-limit gating
    instead of blanking `assrt` / `subdl` credentials inside the working volume
- The pulled FnOS config can enable expensive paths such as timeline fixing.
  The local candidate now skips audio-fallback timeline fixing for long media
  when no usable embedded subtitle exists, so subtitle download completion is
  not blocked by a full-movie VAD pass.
- In practice, `subhd` daily download limits can block the first native-Chinese
  round and prevent the full profile from reaching the LLM rounds.
- `scripts/local_llm_acceptance.ps1` exists so the LLM fallback path can still
  be revalidated on the same candidate/container/root without being coupled to
  that daily-limit gate.

## Current Residue State

Retained report-side files:

- `D:\tmp\csf-local-candidate\reports\20260619-195837-520-e2e-matrix\residue-audit-20260619-200219-896.json`
- `D:\tmp\csf-local-candidate\reports\20260619-195745-556\residue-audit-20260619-195806-948.json`
- `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260619-181953-522.json`
- `D:\tmp\csf-local-candidate\reports\sample-pool-audit-20260619-053556-615.json`

Retained runtime keep-set:

- container: `chinesesubfinder-local-candidate`
- image: `chinesesubfinder:local-candidate`
- immutable baseline volumes:
  - `csf_fnos_config_full_20260619`
  - `csf_fnos_browser_full_20260619`
- active working volumes:
  - `csf_fnos_config_working`
  - `csf_fnos_browser_working`
- sample spec pool: `D:\tmp\csf-real-media-stage`
- current route-coverage evidence window only

## Latest Provider Findings

- `assrt` movie native Chinese passed on clean working volume:
  - `D:\tmp\csf-local-candidate\reports\20260619-153813-522-e2e-matrix\e2e-summary.json`
- `opensubtitles` movie native Chinese re-passed after the real FnOS movie
  mount was brought into the container at startup:
  - `D:\tmp\csf-local-candidate\reports\20260619-164307-305-e2e-matrix\e2e-summary.json`
- `assrt` series native Chinese re-passed after the harness stopped wiping the
  local full-config credentials during route-setting updates:
  - `D:\tmp\csf-local-candidate\reports\20260619-165256-310-e2e-matrix\e2e-summary.json`
- `assrt` series native Chinese also re-passed after the route-isolation fix
  that preserves supplier credentials in the working volume:
  - `D:\tmp\csf-local-candidate\reports\20260619-180927-366-e2e-matrix\e2e-summary.json`
- `assrt` series native Chinese also re-passed after the default-chain
  restructuring and image rebuild:
  - `D:\tmp\csf-local-candidate\reports\20260619-183251-124-e2e-matrix\e2e-summary.json`
- `opensubtitles` movie native Chinese passed on clean working volume:
  - `D:\tmp\csf-local-candidate\reports\20260619-154236-887-e2e-matrix\e2e-summary.json`
- `opensubtitles` movie English fallback also passed after the full FnOS config
  was re-pulled into the local baseline and re-adapted for local runtime:
  - `D:\tmp\csf-local-candidate\reports\20260619-195837-520-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=movie.english_fallback`,
    `final_output_has_chinese=false`
- `assrt` series native Chinese also passed after the full FnOS config was
  re-pulled again and re-adapted for local runtime:
  - `D:\tmp\csf-local-candidate\reports\20260619-200709-664-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=series.native_chinese`,
    `final_output_has_chinese=true`
- `subhd` series native Chinese also passed after the full FnOS config was
  re-pulled again, still using local `ddddocr` plus the native SubHD gate:
  - `D:\tmp\csf-local-candidate\reports\20260619-201239-517-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=series.native_chinese`,
    `final_output_has_chinese=true`
- `subtitlecat_translated` explicit translated-Chinese route passed again after
  the harness bug was fixed so the round now isolates SubtitleCat correctly:
  - `D:\tmp\csf-local-candidate\reports\20260619-202802-077-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`,
    `route_key=series.subtitlecat_translated`,
    `final_output_has_chinese=true`
- `subtitlecat_translated` explicit translated-Chinese route also passed after
  the new `actual_supplier` assertion was added:
  - `D:\tmp\csf-local-candidate\reports\20260619-212036-542-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`,
    `route_key=series.subtitlecat_translated`,
    `actual_supplier=subtitlecat_translated`,
    `final_output_has_chinese=true`
- `subtitlecat` default English fallback also passed again on the same sample
  under the current pulled FnOS config:
  - `D:\tmp\csf-local-candidate\reports\20260619-203110-654-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=series.english_fallback`,
    `final_output_has_chinese=false`
- `subtitlecat` default English fallback also passed after the new
  `actual_supplier` assertion was added:
  - `D:\tmp\csf-local-candidate\reports\20260619-211716-624-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=series.english_fallback`,
    `actual_supplier=subtitlecat`, `final_output_has_chinese=false`
- `moviesubtitles` English fallback also re-passed after the full FnOS config
  was re-pulled again:
  - `D:\tmp\csf-local-candidate\reports\20260619-201535-498-e2e-matrix\e2e-summary.json`
  - result: `job_terminal_status=3`, `route_key=movie.english_fallback`,
    `final_output_has_chinese=false`
- `subhd` movie native Chinese passed on clean working volume with local gate
  and no forced external OCR:
  - `D:\tmp\csf-local-candidate\reports\20260619-154351-895-e2e-matrix\e2e-summary.json`
- `subtitlecat` explicit translated-Chinese route passed on clean working
  volume:
  - `D:\tmp\csf-local-candidate\reports\20260619-154527-470-e2e-matrix\e2e-summary.json`
- `subtitlecat` default English fallback route passed on clean working volume:
  - `D:\tmp\csf-local-candidate\reports\20260619-154739-953-e2e-matrix\e2e-summary.json`
- `subtitlecat` explicit translated-Chinese route also re-passed on a real
  mounted series sample:
  - `D:\tmp\csf-local-candidate\reports\20260619-165720-000-e2e-matrix\e2e-summary.json`
- `subtitlecat` default English fallback route also re-passed on a real mounted
  series sample:
  - `D:\tmp\csf-local-candidate\reports\20260619-170035-844-e2e-matrix\e2e-summary.json`
- `subdl` route findings were tightened again after fixing the matrix route
  classifier so `AcceptNoSubFound` no longer mislabels successful English
  fallback as `safe_fail`:
  - code fix:
    - `scripts/local_e2e_matrix.ps1`
  - current serial movie proof:
    - `D:\tmp\csf-local-candidate\reports\20260619-181417-903-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=movie.english_fallback`,
      `final_output_has_chinese=false`
  - current serial series proof:
    - `D:\tmp\csf-local-candidate\reports\20260619-180008-382-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=3`, `route_key=series.english_fallback`,
      `final_output_has_chinese=false`
  - latest isolated current-config series proof:
    - `D:\tmp\csf-local-candidate\reports\20260619-201123-934-e2e-matrix\e2e-summary.json`
    - result: `job_terminal_status=2`, `route_key=series.safe_fail`,
      `job_error_info=No Sub Found`
    - log detail:
      - the real `subdl` key was used in the request, but the provider still
        ended with no usable subtitle artifact for this episode
  - current conclusion:
    - `subdl` is not a stable native-Chinese primary source on the sampled
      titles
    - both sampled titles currently fall through to English-only output
      instead of producing usable Chinese subtitles
- mixed English fallback chain re-pass after the default-chain restructuring:
  - movie:
    - `D:\tmp\csf-local-candidate\reports\20260619-183355-308-e2e-matrix\e2e-summary.json`
    - result: `route_key=movie.english_fallback`
  - series:
    - `D:\tmp\csf-local-candidate\reports\20260619-183500-434-e2e-matrix\e2e-summary.json`
    - result: `route_key=series.english_fallback`
- clean movie safe-fail proof remains:
  - `D:\tmp\csf-local-candidate\reports\20260619-181812-104-e2e-matrix\e2e-summary.json`
  - result: `route_key=movie.safe_fail`, `job_error_info=No Sub Found`

## Remaining Work

The goal is not complete yet.

Still missing before the full local acceptance objective can be claimed done:

- decide whether `subdl` should stay anywhere in the default route policy, or
  be constrained to a narrower English-fallback role only
- broader low-cost title sampling if a wider confidence window is required

## 2026-06-20 Harness Hardening Addendum

- `scripts/refresh_fnos_working_volumes.ps1` now fixes ownership on the
  refreshed working volumes before the next test round:
  - it now leaves `ChineseSubFinderSettings.json` as `1026:100` instead of
    `root:root`
  - live verification after refresh without restarting the container:
    - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-044041-605`
    - proves the running container could still log in and probe `subdl`
- `scripts/local_candidate_round.ps1` no longer pollutes adapted settings with
  ordered-dictionary container members such as `Count`, `Keys`, and `Values`
  during local runtime normalization:
  - before the fix, that merge bug could remove top-level
    `subtitle_sources.subtitlecat_settings` from the adapted shadow file and
    make `local_e2e_matrix.ps1` fail before job execution
  - after the fix, the adapted shadow file again contains the expected
    `subtitlecat_settings.enabled` structure
- `scripts/local_e2e_matrix.ps1` now records route-policy warnings when the
  requested supplier set does not match the backend wiring:
  - current proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-044610-120-e2e-matrix`
  - proves:
    - `policy_warnings.json` explicitly records that `tvsubtitles` is not wired
      into the backend default English fallback chain
    - the round now fails for the real business reason
      `job_error_info=No Sub Found`, not because the harness crashed

## 2026-06-20 Full Non-LLM Acceptance Re-run After Harness Fixes

- with the refreshed harness and the same pulled FnOS working volumes, the full
  non-LLM local acceptance profile re-passed end to end on candidate image
  `cd2dd11cfc8f`
- latest route proofs from that rerun:
  - `movie.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-045337-821-e2e-matrix`
    - `actual_supplier=subhd`
  - `movie.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-045436-321-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-045708-901-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `movie.safe_fail`
    - `D:\tmp\csf-local-candidate\reports\20260620-045800-023-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-045838-216-e2e-matrix`
    - `actual_supplier=subhd`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-050123-229-e2e-matrix`
    - `actual_supplier=subtitlecat`
  - `series.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-050358-181-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
- route coverage snapshot after the rerun:
  - `D:\tmp\csf-local-candidate\reports\route-coverage-snapshot-20260620-050835-197.json`
  - result:
    - `present_route_count=10`
    - `missing_required_route_count=0`
    - `coverage_ok=true`
- cleanup and residue state after pruning stale reports:
  - `D:\tmp\csf-local-candidate\reports\residue-audit-20260620-050909-387.json`
  - result:
    - only one active local candidate container remains
    - only one active local candidate image remains
    - only the active candidate root plus the two active sample pools remain in
      `D:\tmp`

## 2026-06-20 Default All-Supplier Reality Check

- after the clean full non-LLM rerun, two extra default-state rounds were run
  without narrowing the supplier set, using the pulled FnOS config as-is:
  - movie default all-supplier round:
    - `D:\tmp\csf-local-candidate\reports\20260620-051046-861-e2e-matrix`
    - result:
      - `route_key=movie.native_chinese`
      - `actual_supplier=subhd`
  - series default all-supplier round:
    - `D:\tmp\csf-local-candidate\reports\20260620-051237-227-e2e-matrix`
    - result:
      - `route_key=series.native_chinese`
      - `actual_supplier=opensubtitles`
- this matters for chain design:
  - the default full-source runtime is not collapsing onto one hard-coded
    supplier
  - movie Chinese-first still naturally resolves to `subhd` on the sampled
    title
  - series Chinese-first can legitimately resolve to `opensubtitles` first when
    it produces the winning usable Chinese subtitle
- latest supplier health snapshot on the same default-state runtime:
  - `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-051618-724`
  - summary:
    - valid and enabled:
      `xunlei`, `shooter`, `assrt`, `subdl`, `opensubtitles`, `tvsubtitles`,
      `moviesubtitles`, `subtitlecat`, `subhd`
    - invalid:
      `subtitle_best` (`reason=disabled`)

## 2026-06-20 Route-Stage Evidence Hardening

- the local runtime was rebuilt as candidate image `c946b859fc90`
- the downloader now logs an explicit `SubtitleRouteStage` marker when a sample
  is finalized through:
  - `primary_chinese`
  - `translated_chinese`
  - `llm_fallback`
  - `english_fallback`
  - `safe_fail`
- `scripts/local_e2e_matrix.ps1` now captures that stage evidence into
  `supplier-evidence.json` and prefers it over content-only heuristics when
  writing `route_key`
- current proof that the stage evidence is working on the rebuilt image:
  - isolated movie `subdl` round:
    - `D:\tmp\csf-local-candidate\reports\20260620-054350-148-e2e-matrix`
    - proves:
      - `actual_supplier=subdl`
      - `route_stage=english_fallback`
      - `route_key=movie.english_fallback`
  - isolated movie `opensubtitles` round:
    - `D:\tmp\csf-local-candidate\reports\20260620-054437-964-e2e-matrix`
    - proves:
      - `actual_supplier=opensubtitles`
      - `route_stage=primary_chinese`
      - `route_key=movie.native_chinese`
- this settled the remaining ambiguity around `opensubtitles`:
  - when `opensubtitles` is enabled, the current backend policy still registers
    it in the primary Chinese chain as well as the English fallback chain
  - so `opensubtitles` cannot be treated as an English-only isolated supplier
    under the current runtime policy
- the harness now warns about that coupling explicitly:
  - `D:\tmp\csf-local-candidate\reports\20260620-054642-873-e2e-matrix`
  - `policy_warnings.json` now records:
    - requested English fallback supplier `opensubtitles` also participates in
      the backend primary Chinese chain when enabled

## 2026-06-20 Explicit Stage-Order Refactor And Active-Image Re-Verification

- the supplier-plan builder no longer relies on one shared site sequence to
  incidentally drive every route stage
- the runtime policy is now expressed with explicit per-stage ordering:
  - primary Chinese stage:
    - `assrt`, `subhd`, `shooter`, `xunlei`, `opensubtitles`
  - default English fallback stage:
    - `opensubtitles`, `subdl`, `subtitlecat`, `moviesubtitles`
  - translated-Chinese fallback stage:
    - `subtitlecat_translated`
- this was implemented as an architectural refactor, not a behavior change:
  - `pkg/types/common/sub_site_sequence.go`
  - `pkg/logic/pre_download_process/pre_download_proces.go`
  - new tests in:
    - `pkg/types/common/sub_site_sequence_test.go`
    - `pkg/logic/pre_download_process/pre_download_proces_test.go`
- targeted verification passed after the refactor:
  - `go test ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/downloader -count=1`

## 2026-06-20 Full Non-LLM Re-Run On Current Active Image

- after the stage-order refactor, the full local non-LLM acceptance suite was
  re-run again and passed on the current active candidate image
  `beb997acf572`
- current route proof set:
  - `movie.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-061444-480-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `movie.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-062033-929-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-062210-684-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_stage=english_fallback`
  - `movie.safe_fail`
    - `D:\tmp\csf-local-candidate\reports\20260620-062243-708-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-062311-276-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-062409-742-e2e-matrix`
    - `actual_supplier=subtitlecat`
    - `route_stage=english_fallback`
  - `series.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-062448-272-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`

## 2026-06-20 Default English-Fallback Competition On Current Active Image

- after the refactor and full re-run, the default English fallback competition
  was re-checked again on the same active image `beb997acf572`
- movie competition, with primary Chinese explicitly disabled and only
  `subdl`, `subtitlecat`, and `moviesubtitles` allowed:
  - `D:\tmp\csf-local-candidate\reports\20260620-062835-621-e2e-matrix`
  - result:
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=movie.english_fallback`
- series competition, with primary Chinese explicitly disabled and only
  `subdl` and `subtitlecat` allowed:
  - `D:\tmp\csf-local-candidate\reports\20260620-062924-525-e2e-matrix`
  - result:
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=series.english_fallback`
- movie-only tail fallback still remains independently usable:
  - `D:\tmp\csf-local-candidate\reports\20260620-060341-454-e2e-matrix`
  - result:
    - `actual_supplier=moviesubtitles`
    - `route_stage=english_fallback`
- current chain conclusion from those competition rounds:
  - keep `subdl` ahead of `subtitlecat` in the default English fallback order
  - keep `subtitlecat` enabled by default as the broader downstream fallback
  - keep `moviesubtitles` as the movie-only tail fallback
  - no evidence currently justifies moving `subtitlecat` ahead of `subdl`

## 2026-06-20 Supplier Re-Probe And JSON Artifact Hardening

- `subdl` was re-proved again under true isolation from the primary Chinese
  chain on the same active image:
  - movie:
    - `D:\tmp\csf-local-candidate\reports\20260620-063728-937-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=movie.english_fallback`
  - series:
    - `D:\tmp\csf-local-candidate\reports\20260620-063804-363-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=series.english_fallback`
- `tvsubtitles` was re-checked in isolation again on a real mounted-library
  sample:
  - `D:\tmp\csf-local-candidate\reports\20260620-063847-623-e2e-matrix`
  - result:
    - `job_error_info=No Sub Found`
    - `route_key=series.safe_fail`
    - `policy_warnings.json` explicitly states that `tvsubtitles` is not wired
      into the backend default English fallback chain
- `subtitle_best` was re-confirmed as a two-role dependency in code, not just
  an old optional supplier toggle:
  - direct supplier path:
    - `pkg/logic/sub_supplier/subtitle_best/subtitle_best.go`
  - shared service path:
    - `pkg/subtitle_best_api/subtitle_best_api.go`
    - `pkg/media_info_dealers/dealers.go`
    - `pkg/logic/sub_supplier/subhd/code_provider.go`
  - current pulled FnOS working config still has:
    - `subtitle_best_settings.enabled=false`
    - `subtitle_best_settings.api_key=""`
  - therefore the supplier role still remains runtime-unverified, while the
    shared-service role is confirmed present in architecture
- the local acceptance/reporting scripts were also hardened so bad control
  characters or broken surrogate pairs in sample names, subtitle samples, or
  log lines no longer poison JSON report artifacts:
  - implementation:
    - `scripts/local_acceptance_matrix_utils.ps1`
    - `scripts/local_e2e_matrix.ps1`
    - `scripts/local_candidate_round.ps1`
    - `scripts/local_materialize_sample_pool.ps1`
    - `scripts/local_sample_pool_audit.ps1`
    - `scripts/local_supplier_status_snapshot.ps1`
  - post-hardening live proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-064523-788-e2e-matrix`
    - `D:\tmp\csf-local-candidate\reports\20260620-064711-526-e2e-matrix`
  - these new reports are UTF-8 JSON and can be parsed again by
    `ConvertFrom-Json` without the earlier malformed-summary failure mode

## 2026-06-20 Full Non-LLM Re-Run After Acceptance-Matrix Contract Upgrade

- the acceptance matrix itself was tightened so the default English fallback
  path is no longer merely "some English fallback happened":
  - `scripts/local_acceptance_matrix.psd1`
  - `scripts/local_acceptance_runner.ps1`
  - `scripts/local_candidate_round.ps1`
  - `scripts/local_e2e_matrix.ps1`
- the new contract explicitly requires the default English fallback winner to
  remain `subdl`:
  - movie default chain:
    - suppliers requested: `subdl`, `subtitlecat`, `moviesubtitles`
    - expected winner: `subdl`
  - series default chain:
    - suppliers requested: `subdl`, `subtitlecat`
    - expected winner: `subdl`
- targeted proof for that contract passed before the full rerun:
  - movie:
    - `D:\tmp\csf-local-candidate\reports\20260620-065216-314-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_key=movie.english_fallback`
  - series:
    - `D:\tmp\csf-local-candidate\reports\20260620-065216-762-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_key=series.english_fallback`
- after that contract upgrade, the full local non-LLM acceptance suite was
  re-run again and passed on the rebuilt active candidate image
  `2da7ffb59123`
- build and targeted verification also passed in that rerun:
  - frontend production build:
    - `D:\tmp\csf-local-candidate\reports\20260620-065430-096\frontend-build.log`
  - targeted Go test slice:
    - `D:\tmp\csf-local-candidate\reports\20260620-065430-096\go-targeted-tests.log`
- current proof set from that rerun:
  - `movie.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-065900-907-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `movie.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-070004-943-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
  - `movie.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-070139-977-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
  - `movie.safe_fail`
    - `D:\tmp\csf-local-candidate\reports\20260620-070209-868-e2e-matrix`
    - `job_error_info=No Sub Found`
  - `series.native_chinese`
    - `D:\tmp\csf-local-candidate\reports\20260620-070238-997-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
  - `series.english_fallback`
    - `D:\tmp\csf-local-candidate\reports\20260620-070354-212-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
  - `series.subtitlecat_translated`
    - `D:\tmp\csf-local-candidate\reports\20260620-070433-659-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
- current chain conclusion after the stricter rerun remains:
  - keep `subdl` ahead of `subtitlecat` in the default English fallback chain
  - keep `subtitlecat` as the broader downstream English provider and explicit
    translated-Chinese fallback
  - keep `moviesubtitles` as the movie-only tail English fallback

## 2026-06-20 Subtitle Content Sample Audit

- route correctness is no longer the only proved layer; representative output
  files from the latest non-LLM rerun were also spot-checked for filename
  match, language expectation, and obvious wrong-title /乱码 / empty-output risk
- sampled outputs:
  - native Chinese movie via `subhd`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-065900-907-e2e-matrix\耐撕侦探 (2016) - 2160p.chinese(简).default.srt`
    - spot result:
      - filename matches target movie
      - output is real Simplified Chinese subtitle text, not wrong-title noise
      - opening lines are plausible movie dialogue / title-card content
  - translated Chinese movie via `subtitlecat_translated`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-070004-943-e2e-matrix\星际穿越 (2014) - 2160p.chinese(简).default.srt`
    - spot result:
      - Chinese lines align with the known opening exchange of the English
        fallback sample
      - no empty output or obvious cross-title mismatch was observed
  - English fallback movie via `subdl`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-070139-977-e2e-matrix\星际穿越 (2014) - 2160p.chinese(英).default.srt`
    - spot result:
      - output is English as expected for the fallback stage
      - opening lines match the same scene as the translated-Chinese sample
      - this reduces the risk that the route only "downloaded something" but
        attached the wrong subtitle body
  - native Chinese / bilingual series via `subhd`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-070238-997-e2e-matrix\黑袍纠察队 - S01E03 - 第 3 集.chinese(简英).default.ass`
    - spot result:
      - bilingual ASS dialogue lines are present
      - episode naming matches the target sample
      - content structure looks like a normal matched episode subtitle, not a
        damaged container or unrelated file
  - English fallback series via `subdl`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-070354-212-e2e-matrix\拥挤的房间 - S01E02 - 第 2 集.chinese(英).default.srt`
    - spot result:
      - output is English as expected
      - dialogue content is plausible for a normal episode subtitle and not
        obviously mismatched
  - translated Chinese series via `subtitlecat_translated`
    - file:
      `D:\tmp\csf-local-candidate\reports\20260620-070433-659-e2e-matrix\洛佩兹一家 - S01E01 - 第 1 集.chinese(简).default.srt`
    - spot result:
      - Chinese output is present and understandable
      - however the lines still show machine-translation stiffness / rough line
        wrapping compared with native-Chinese sources
- current interpretation from this sample audit:
  - no sampled output showed an obvious wrong-title attach, empty-body success,
    or gross language-stage mismatch
  - native-Chinese routes currently look materially healthier than translated
    fallback routes
  - `subtitlecat_translated` is usable as an explicit translated fallback, but
    its content quality should still be described as machine-translation-grade,
    not human-polish-grade

## 2026-06-20 Supplier Health vs Route Eligibility Recheck

- a fresh live supplier snapshot was taken on the current active candidate
  runtime after the stricter non-LLM rerun:
  - report:
    `D:\tmp\csf-local-candidate\reports\supplier-status-20260620-080359-173`
  - live probe result:
    - `subdl valid=true`
    - `tvsubtitles valid=true`
    - `subtitlecat valid=true`
    - `subtitle_best valid=false reason=disabled`
- the snapshot script was hardened so it no longer reports probe health without
  policy context:
  - `scripts/local_supplier_status_snapshot.ps1` now annotates each supplier
    with:
    - `participates_in_primary_chain`
    - `participates_in_default_english_fallback`
    - `participates_in_translated_chinese_fallback`
    - `policy_state`
    - `policy_note`
  - this directly resolves the earlier ambiguity where `tvsubtitles valid=true`
    could be misread as "part of the real default fallback chain"
- the same runtime was then re-tested with an isolated `tvsubtitles`-only
  series English-fallback round:
  - round:
    `D:\tmp\csf-local-candidate\reports\20260620-080111-540-e2e-matrix`
  - hard result:
    - `local-e2e-matrix.log` warns that requested supplier `tvsubtitles` is not
      wired into the backend default English fallback chain
    - terminal result stayed `No Sub Found`
- updated interpretation:
  - `tvsubtitles` is currently reachable enough to pass `CheckAlive`
  - that does not upgrade it into a usable default-chain supplier on the
    current backend policy
  - current practical chain conclusion remains unchanged:
    - `subdl` is the real default English fallback
    - `subtitlecat` remains downstream fallback and explicit translated-Chinese
      fallback
  - `tvsubtitles` stays probeable only and should not be described as part of
      the active fallback route

## 2026-06-20 LLM Audit and Log Cleanup Tightening

- the subtitle content audit script now includes the LLM fallback routes by
  default:
  - `scripts/local_subtitle_content_audit.ps1`
  - latest report:
    `D:\tmp\csf-local-candidate\reports\subtitle-content-audit-20260620-081447-692.json`
- the audit now records:
  - `english_line_ratio`
  - `english_only_line_ratio`
  - route-stage fallback display for LLM routes
  - warnings when translated fallback output still leaves noticeable
    untranslated English
- current LLM sample findings on the retained local evidence:
  - movie LLM fallback:
    - `20260620-022515-500-e2e-matrix`
    - `route_stage=llm_fallback`
    - `english_only_line_count=31 / dialogue_line_count=2936`
    - still operational, but some untranslated English remains
  - series LLM fallback:
    - `20260620-024514-233-e2e-matrix`
    - `route_stage=llm_fallback`
    - `english_only_line_count=59 / dialogue_line_count=870`
    - this sample is materially rougher than the movie sample
  - comparison point:
    - `subtitlecat_translated` movie sample no longer triggers the stricter
      untranslated-English warning:
      - `english_only_line_count=4 / dialogue_line_count=2741`
- interpretation:
  - the LLM fallback chain is operational and can finish full subtitle output
  - it is not yet as linguistically clean as the best native-Chinese routes
  - the current remaining weakness is no longer "cannot run"; it is "still
    leaves untranslated English in some translated output, especially on the
    retained series sample"
  - cleanup was also tightened for LLM task garbage inside the candidate root:
  - `scripts/local_residue_audit.ps1` now tracks old
    `config-prepull-snapshot\llm-logs\*` task directories
  - `scripts/local_cleanup.ps1` can now safely delete only those approved
    candidate-internal LLM task paths
  - this round reduced retained LLM task directories from `8` to `2`
  - total retained LLM task-log size dropped from about `7.83 MB` to
    `2.31 MB`

## 2026-06-20 LLM Series Fallback Repair Pass

- `third_party/subflow/src/subflow/translate_job.py` was tightened in two
  ways:
  - a targeted repair pass now re-asks the model only for cues that still look
    like untranslated English dialogue
  - when post-processing intentionally drops pure noise / watermark text, the
    renderer no longer falls back to the original English source cue
- static verification after this change passed:
  - `python -m py_compile third_party\subflow\src\subflow\translate_job.py`
  - `go test ./pkg/llm_subtitle_fallback ./pkg/downloader -count=1`
- live verification was then repeated twice on the same real-library LLM series
  sample (`crowded-room-s01e03.json`) with the rebuilt candidate image:
  - first repaired rerun:
    - `20260620-082931-240-e2e-matrix`
    - `english_only_line_count=25 / dialogue_line_count=877`
  - second rerun after fixing the empty-postprocess fallback path:
    - `20260620-084436-792-e2e-matrix`
    - `english_only_line_count=16 / dialogue_line_count=863`
    - no untranslated-English warning is emitted by the current audit
- concrete direction of improvement on the retained series sample:
  - before this repair work:
    - retained older series LLM sample was `59 / 870`
  - after the first repair pass:
    - `25 / 877`
  - after the empty-fallback fix:
    - `16 / 863`
- the remaining English-only residue on the newest retained series sample is no
  longer whole conversational lines; it is now dominated by standalone speaker
  labels such as:
  - `[Matty]`
  - `[Rya]`
  - `[Danny]`
  - `[Annabelle]`
  - `[Angelo]`
  - `[Ariana]`
- current interpretation:
  - the LLM fallback chain is now materially cleaner on the retained series
    sample than it was at the start of this round
  - the main remaining polish gap is speaker-label normalization, not broad
    untranslated English dialogue leakage

## 2026-06-20 LLM Speaker-Label Normalization Follow-up

- `third_party/subflow/src/subflow/translate_job.py` was tightened again so
  the repair pass no longer treats raw English speaker labels as acceptable:
  - bare labels like `[Matty]` now still trigger repair
  - mixed label lines like `[Rya] 我去了...` now also trigger repair instead of
    being accepted just because the line already contains Chinese
  - the prompt was updated to explicitly forbid raw English-only or mixed
    English-leading speaker-label forms
- focused Python verification was added and passed:
  - `python -m unittest subflow.test_translate_job subflow.test_openai_compatible_client`
- live rebuilt-image verification was then repeated twice on the same retained
  real-library LLM series sample:
  - after the first label-only tightening:
    - `20260620-090352-653-e2e-matrix`
    - `english_only_line_count=7 / dialogue_line_count=866`
    - sample output now includes normalized labels such as `【马蒂】`
  - after the mixed-label repair trigger:
    - `20260620-091857-671-e2e-matrix`
    - `english_only_line_count=4 / dialogue_line_count=867`
    - mixed-language lines also dropped to `22`
    - sample output now includes normalized labels such as `[莉亚] 我去过...`
- updated retained improvement ladder on the same series sample:
  - older retained baseline:
    - `59 / 870`
  - first repair pass:
    - `25 / 877`
  - empty-postprocess fallback fix:
    - `16 / 863`
  - speaker-label-only repair gate:
    - `7 / 866`
  - mixed-label repair gate:
    - `4 / 867`
- current interpretation:
  - the LLM fallback route is now no longer leaking whole English dialogue on
    this retained series sample
  - remaining English is mostly proper names or title fragments embedded in
    otherwise Chinese output, not route-level translation failure

## 2026-06-20 Cross-sample LLM Verification and Fragment Tightening

- after the speaker-label work, the same local mounted-library runtime was
  checked on additional real samples without changing the FnOS-backed config:
  - series sample:
    - `the-boys-s01e03.json`
    - retained round: `20260620-093247-472-e2e-matrix`
    - result:
      - `route_key=series.llm_fallback`
      - `actual_supplier=subtitlecat`
      - `english_only_line_count=8 / dialogue_line_count=1293`
      - `mixed_language_line_count=82`
    - interpretation:
      - no warning threshold hit
      - remaining English is dominated by names and in-universe terms such as
        `Translucent`, `Homelander`, `A-Train`, `Compound V`
  - movie sample:
    - initial post-label-normalization movie rerun:
      - `20260620-094608-548-e2e-matrix`
      - `english_only_line_count=17 / dialogue_line_count=2939`
    - after explicitly forcing repair of one-letter OCR fragments:
      - `20260620-101525-600-e2e-matrix`
      - `english_only_line_count=9 / dialogue_line_count=2938`
    - after additionally nudging standalone call-names toward Chinese
      transliteration:
      - `20260620-104213-358-e2e-matrix`
      - `english_only_line_count=7 / dialogue_line_count=2942`
      - `mixed_language_line_count=109`
- the movie residual set on the newest retained round is now narrow and
  intelligible:
  - `NASA`
  - `Lazarus`
  - `CASE`
  - `RPM`
  - `TARS`
- important interpretation from this cross-sample pass:
  - the recent LLM cleanup was not limited to `crowded-room-s01e03`
  - the route now behaves acceptably on both an additional series sample and a
    long movie sample under the same mounted-library local Docker runtime
  - current remaining English is largely domain terminology and proper names,
    not broad untranslated sentence leakage

## 2026-06-20 Supplier-role Recheck and Isolation Evidence Retention

- a fresh live supplier snapshot on the current candidate runtime confirmed:
  - `subdl`: valid, default English fallback only
  - `subtitlecat`: valid, default English fallback only
  - `moviesubtitles`: valid, movie-only tail English fallback
  - `opensubtitles`: valid, participates in both the primary Chinese chain and
    English fallback policy
  - `tvsubtitles`: valid on health probe only, still not wired into the active
    backend default English fallback chain
  - `subtitle_best`: still disabled in the pulled FnOS config, so supplier-role
    runtime proof is still absent
- two new supplier-level runtime checks were then added on the real
  mounted-library movie sample (`interstellar-2014.json`):
  - `opensubtitles` requested as the only English fallback supplier:
    - round: `20260620-111139-680-e2e-matrix`
    - actual outcome:
      - `actual_supplier=opensubtitles`
      - `route_stage=primary_chinese`
      - `route_key=movie.native_chinese`
    - interpretation:
      - this is not an English-fallback isolation success
      - it is direct proof that `opensubtitles` is genuinely active in the
        backend primary Chinese chain under the current runtime policy
  - `moviesubtitles` requested as the isolated English fallback supplier:
    - round: `20260620-111318-314-e2e-matrix`
    - outcome:
      - `actual_supplier=moviesubtitles`
      - `route_stage=english_fallback`
      - `route_key=movie.english_fallback`
      - `route_assertion=passed`
- the local cleanup policy was also tightened so that supplier-isolation
  evidence is no longer discarded as if it were just a duplicate route pass:
  - `scripts/local_residue_audit.ps1` now retains:
    - the normal route-level kept rounds
    - plus the latest distinct supplier-isolation proof per route
    - plus two recent policy-warning e2e rounds instead of only one
  - this matters because otherwise the new `opensubtitles` and
    `moviesubtitles` proofs would be silently deleted during routine cleanup

## 2026-06-20 Safe-fail Evidence Retention Repair

- the next cleanup gap surfaced on earlier `shooter` / `xunlei` safe-fail
  rounds: when a run ended in `No Sub Found`, the old `e2e-summary.json` did
  not preserve enough request context for later residue grouping
- the fix was intentionally narrow:
  - `scripts/local_e2e_matrix.ps1`
    - now writes:
      - `requested_primary_suppliers`
      - `requested_english_fallback_suppliers`
      - `requested_isolation_supplier`
      - `expected_route_key`
  - `scripts/local_residue_audit.ps1`
    - now uses `expected_route_key` when `route_key` is absent
    - now uses `requested_isolation_supplier` when `actual_supplier` is absent
- this was immediately re-verified on the same mounted-library movie sample
  `interstellar-2014.json` using config volume `csf_fnos_config_working`:
  - `20260620-113450-327-e2e-matrix`
    - requested isolated supplier: `xunlei`
    - result:
      - `job_terminal_status=2`
      - `job_error_info=No Sub Found`
      - `requested_isolation_supplier=xunlei`
      - `expected_route_key=movie.safe_fail`
      - `route_key=movie.safe_fail`
      - `checks.no_sub_safe_failure=passed`
  - `20260620-113548-319-e2e-matrix`
    - requested isolated supplier: `shooter`
    - result:
      - `job_terminal_status=2`
      - `job_error_info=No Sub Found`
      - `requested_isolation_supplier=shooter`
      - `expected_route_key=movie.safe_fail`
      - `route_key=movie.safe_fail`
      - `checks.no_sub_safe_failure=passed`
- the retention policy then proved the fix worked:
  - latest kept route-level safe-fail proof:
    - `20260620-113548-319-e2e-matrix`
  - extra kept supplier-isolation proof for the same route:
    - `20260620-113450-327-e2e-matrix`
    - reason:
      `retained supplier isolation evidence for route movie.safe_fail via xunlei`
- cleanup was executed from the fresh manifest immediately after verification:
  - deleted stale duplicate directories:
    - `20260620-112428-777-e2e-matrix`
    - `20260620-112236-896-e2e-matrix`
    - `20260620-070209-868-e2e-matrix`
  - post-cleanup spot check confirmed each path no longer existed while the
    active container `chinesesubfinder-local-candidate` remained running

## 2026-06-20 Post-cleanup Route Snapshot Refresh

- after the cleanup and retention fixes, one more focused series-side
  `safe_fail` verification was added so the newer summary schema is now proven
  on both movie and series workloads
- live round:
  - `20260620-114153-260-e2e-matrix`
  - sample:
    - `the-boys-s01e02.json`
    - `黑袍纠察队 - S01E02 - 第 2 集`
  - requested isolated supplier:
    - `xunlei`
  - result:
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
    - `requested_isolation_supplier=xunlei`
    - `expected_route_key=series.safe_fail`
    - `route_key=series.safe_fail`
    - `checks.no_sub_safe_failure=passed`
- route coverage snapshot was then regenerated from the current retained proof
  set:
  - `route-coverage-snapshot-20260620-114233-192.json`
  - result:
    - `coverage_ok=true`
    - `missing_required_route_count=0`
    - required routes present:
      - `movie.native_chinese`
      - `movie.subtitlecat_translated`
      - `movie.english_fallback`
      - `movie.safe_fail`
      - `movie.llm_fallback`
      - `series.native_chinese`
      - `series.subtitlecat_translated`
      - `series.english_fallback`
      - `series.llm_fallback`
    - retained extra route proof also present:
      - `series.safe_fail`
- a fresh supplier-policy snapshot was also regenerated after cleanup:
  - `supplier-status-20260620-114124-315`
  - stable conclusions remained unchanged:
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
    - policy warning still explicit:
      - `tvsubtitles` is valid on health probe but not wired into the active
        default route chain

## 2026-06-20 Content Audit Refresh and Residual-risk Readout

- `scripts/local_subtitle_content_audit.ps1` was tightened so the audit now
  records:
  - `english_only_samples`
  - `mixed_language_samples`
  - `bilingual_presentation_samples`
  - `looks_bilingual_presentation`
- this removed an earlier blind spot where a legit `简英` dual-language ASS
  track could look artificially “too English-heavy” even though the file was
  behaving as intended
- refreshed audit artifact:
  - `subtitle-content-audit-20260620-114959-484.json`
- current route-level reading from that audit:
  - `movie.english_fallback`
    - behaves as expected English-only fallback
    - sample residuals are ordinary full English dialogue, not corruption
  - `series.english_fallback`
    - behaves as expected English-only fallback
  - `series.native_chinese`
    - now explicitly classed as bilingual presentation
    - `looks_bilingual_presentation=true`
    - `bilingual_presentation_line_ratio=0.977`
    - this is not a regression signal
  - `movie.llm_fallback`
    - remaining pure-English residues are now narrow:
      - `NASA`
      - `Lazarus`
      - `CASE`
      - `RPM`
      - `TARS`
    - mixed-language residues are mostly proper-name and acronym carry-through
  - `series.llm_fallback`
    - remaining pure-English residues are now narrow character / alias terms:
      - `Translucent`
      - `M.M.`
      - `Hughie`
      - `Mother's Milk`
      - `A-Train`
      - `Alek`
  - `movie.subtitlecat_translated`
    - remaining pure-English residues are the isolated OCR-style fragments:
      - `S。`
      - `T。`
      - `A.`
      - `Y。`
  - `movie.native_chinese`
    - residuals on the retained `assrt` proof include:
      - `Gargantua`
      - `Case`
      - `Lois`
      - `S-`
      - `T-`
      - `A-`
      - `Y`
      - `TARS`
- practical interpretation:
  - the chain-level instability problem is now largely solved
  - the main remaining quality debt is concentrated in:
    - proper-name / acronym preservation versus transliteration choices
    - a very small number of OCR-style single-letter fragments
    - not in whole-block untranslated dialogue leakage

## 2026-06-20 LLM Prompt Tightening Recheck

- the LLM prompt was then tightened again, but still only inside the existing
  LLM layer:
  - no new fallback stage was added
  - no new remote dependency was added
  - no route topology was changed
- prompt adjustment scope:
  - recurring spoken character names, codenames, places, missions, and named
    objects are now pushed harder toward standard Chinese rendering
  - raw-form exceptions are narrowed to globally familiar acronyms such as:
    - `NASA`
    - `GPS`
    - `FBI`
    - `CIA`
    - `USB`
    - `AI`
    - `RPM`
- local verification before runtime retest:
  - `python -m py_compile third_party\subflow\src\subflow\translate_job.py`
  - `python -m unittest subflow.test_translate_job subflow.test_openai_compatible_client`
  - `go test ./pkg/logic/pre_download_process ./pkg/downloader -count=1`
  - all passed
- runtime recheck then used the real mounted-library sample
  `the-boys-s01e03.json` on the FnOS-backed local Docker candidate:
  - round:
    - `20260620-115641-939-e2e-matrix`
  - route:
    - `series.llm_fallback`
  - actual supplier:
    - `subtitlecat`
  - result:
    - `job_terminal_status=3`
    - `route_assertion=passed`
    - `llm_output_language=passed`
- refreshed content audit evidence:
  - `subtitle-content-audit-20260620-121020-678.json`
  - this new retained `series.llm_fallback` proof improved from the earlier
    retained round as follows:
    - before:
      - `english_only_line_count=8 / dialogue_line_count=1293`
    - after:
      - `english_only_line_count=2 / dialogue_line_count=1286`
  - new remaining pure-English residue sample set is now down to:
    - `Compound`
    - `M.M.`
  - mixed-language carry-through still exists, but is narrower and more
    name-heavy:
    - `Translucent`
    - `Starlight`
    - `Homelander`
    - `A-Train`
    - `Shockwave`
    - `Seth`
    - `Evan`
    - `Al`
- practical reading:
  - the prompt tightening materially improved real runtime output on the series
    sample
  - the remaining debt is now even more clearly concentrated in translation
    policy choices for names / aliases, not in general sentence translation

## 2026-06-20 Movie-side LLM Recheck After Name Prompt Tightening

- the same tightened LLM policy was then re-verified on the retained movie-side
  real sample `interstellar-2014.json`
- runtime round:
  - `20260620-123803-097-e2e-matrix`
  - route:
    - `movie.llm_fallback`
  - actual supplier:
    - `subtitlecat`
  - result:
    - `job_terminal_status=3`
    - `route_assertion=passed`
    - `llm_output_language=passed`
- refreshed content audit evidence:
  - `subtitle-content-audit-20260620-130350-433.json`
  - movie-side pure-English residue improved again, but by a smaller margin:
    - earlier retained movie proof:
      - `7 / 2942`
    - new retained movie proof:
      - `6 / 2939`
- the structure of the residue also improved:
  - no longer present in the latest pure-English sample set:
    - `Tom!`
    - `Lazarus`
    - `RPM`
  - latest remaining pure-English sample set:
    - `NASA？`
    - `NASA。`
    - `TARS`
    - `CASE！`
    - `CASE！`
    - `TARS？`
- practical interpretation:
  - the latest prompt tightening still helped on the movie sample, but the
    improvement is now incremental rather than large
  - the remaining residue has collapsed almost entirely into policy-level
    choices around whether names like `TARS` / `CASE` and acronym-like terms
    such as `NASA` should stay raw or be forced into Chinese forms
  - this is a much narrower problem than the earlier broad residual leakage

## 2026-06-20 Runtime-image Drift Fix and Fresh LLM Rebuild Proof

- a critical harness finding surfaced after the latest prompt and postprocess
  changes:
  - the local workspace file
    `third_party/subflow/src/subflow/translate_job.py`
    already contained the new named-entity normalization layer, but the active
    candidate container was still running an older embedded subflow runtime
  - direct container inspection proved the drift:
    - old running image before rebuild:
      - `sha256:1cb2d3b9fd20...`
    - old container runtime file:
      - `/opt/subflow/src/subflow/translate_job.py`
      - missing `KNOWN_NAMED_ENTITY_REPLACEMENTS`
      - missing `_normalize_known_named_entities`
- this means the intermediate LLM reruns right before the rebuild were valid as
  route proofs, but they were not valid proofs for the newest
  named-entity/postprocess code path
- the local candidate was then rebuilt and restarted locally before any further
  LLM conclusions were kept:
  - rebuild round:
    - `D:\tmp\csf-local-candidate\reports\20260620-141756-453`
  - new active image:
    - `sha256:8535f02585a6cb51bb42b7edc7e5abc031a7ac7e1cb2581403b8f90ead270d0b`
  - post-rebuild runtime file check inside the container now proves:
    - `KNOWN_NAMED_ENTITY_REPLACEMENTS` exists
    - `_normalize_known_named_entities` exists
    - `Mother(?:'s|’s) Milk -> 母乳` rule exists
- fresh rebuilt-image movie recheck on the same real sample
  `interstellar-2014.json`:
  - round:
    - `20260620-142303-552-e2e-matrix`
  - result:
    - `route_key=movie.llm_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
    - `route_assertion=passed`
    - `llm_output_language=passed`
  - refreshed content audit:
    - `english_only_line_count=3 / dialogue_line_count=2947`
    - `mixed_language_line_count=22`
  - latest pure-English residue is now down to:
    - `NASA`
    - `1G`
  - fresh spot checks on the rebuilt-image output show the intended Chinese
    name normalization is now really live in runtime:
    - `Murph -> 墨菲`
    - `Tom -> 汤姆`
    - `TARS -> 塔斯`
    - `CASE -> 凯斯`
    - `Gargantua -> 卡冈图雅`
- fresh rebuilt-image series recheck on the same real sample
  `the-boys-s01e03.json`:
  - round:
    - `20260620-144314-014-e2e-matrix`
  - result:
    - `route_key=series.llm_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
    - `route_assertion=passed`
    - `llm_output_language=passed`
  - refreshed content audit:
    - `english_only_line_count=0 / dialogue_line_count=1283`
    - `mixed_language_line_count=25`
  - current mixed-language residue is now limited to expected carry-through
    terms such as:
    - `YouTube`
    - `The Seven`
    - `C-4`
    - `Deluca's`
    - `Popclaw`
    - `IPv6`
  - rebuilt-image spot checks show the intended in-universe Chinese rendering
    is now live:
    - `Hughie -> 休伊`
    - `Translucent -> 透明人`
    - `Starlight -> 星光`
    - `A-Train -> 火车头`
    - `Shockwave -> 冲击波`
    - `M.M. / Mother's Milk -> 母乳`
    - `Compound V -> 五号化合物`
- practical interpretation:
  - the earlier suspicion was correct in one specific sense:
    there really was a local runtime/image drift between workspace code and the
    running candidate container
  - after rebuilding the local candidate, the newest LLM cleanup did land in
    the actual tested runtime
  - on rebuilt-image evidence, the LLM fallback route is now materially cleaner
    than the pre-rebuild retained window on both movie and series workloads

## 2026-06-20 Full Non-LLM Recheck On The Same Active Rebuilt Image

- after confirming the active candidate container was really running rebuilt
  image `sha256:8535f02585a6...`, the non-LLM routes were re-run on that same
  live image without another rebuild
- the verification stayed on:
  - image:
    - `chinesesubfinder:local-candidate`
  - container:
    - `chinesesubfinder-local-candidate`
  - config volume:
    - `csf_fnos_config_working`
  - browser volume:
    - `csf_fnos_browser_working`
  - sample pool:
    - `D:\tmp\csf-real-media-runtime\sample-specs`
- current-image non-LLM route proofs:
  - movie native Chinese:
    - `20260620-150459-388-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
    - `route_key=movie.native_chinese`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - movie explicit SubtitleCat translated Chinese:
    - `20260620-150545-259-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
    - `route_key=movie.subtitlecat_translated`
    - `route_assertion=passed`
  - movie default English fallback competition:
    - `20260620-150733-907-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=movie.english_fallback`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - movie safe-fail guard:
    - `20260620-150805-106-e2e-matrix`
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
    - `route_key=movie.safe_fail`
    - `checks.no_sub_safe_failure=passed`
  - series native Chinese:
    - `20260620-150834-373-e2e-matrix`
    - `actual_supplier=subhd`
    - `route_stage=primary_chinese`
    - `route_key=series.native_chinese`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - series default English fallback competition:
    - `20260620-150936-408-e2e-matrix`
    - `actual_supplier=subdl`
    - `route_stage=english_fallback`
    - `route_key=series.english_fallback`
    - `route_assertion=passed`
    - `supplier_assertion=passed`
  - series explicit SubtitleCat translated Chinese:
    - `20260620-151007-088-e2e-matrix`
    - `actual_supplier=subtitlecat_translated`
    - `route_stage=translated_chinese`
    - `route_key=series.subtitlecat_translated`
    - `route_assertion=passed`
- refreshed route coverage snapshot on the same active rebuilt image:
  - `route-coverage-snapshot-20260620-151321-892.json`
  - result:
    - `present_route_count=10`
    - `missing_required_route_count=0`
    - `coverage_ok=true`
- refreshed content audit on the same active rebuilt image:
  - `subtitle-content-audit-20260620-151329-020.json`
  - key route-level reading:
    - `movie.native_chinese`
      - `supplier=subhd`
      - `zh=1889`
    - `movie.english_fallback`
      - `supplier=subdl`
      - `zh=0`, `en=2756`
    - `movie.subtitlecat_translated`
      - `supplier=subtitlecat_translated`
      - `zh=2734`
    - `series.native_chinese`
      - `supplier=subhd`
      - bilingual ASS remains intact
    - `series.english_fallback`
      - `supplier=subdl`
      - `zh=0`, `en=606`
    - `series.subtitlecat_translated`
      - `supplier=subtitlecat_translated`
      - `zh=491`
- refreshed supplier snapshot on the same active rebuilt image:
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
  - invalid / disabled:
    - `subtitle_best`
- practical interpretation:
  - the active rebuilt image `8535f02585a6...` now has both:
    - fresh LLM fallback proof
    - fresh non-LLM route proof
  - ignoring `subtitle_best`, the currently intended default download-chain
    topology is now re-proved on the active local runtime rather than only on
    older retained images

## 2026-06-20 subtitle_best Boundary Regression Tests

- the remaining `subtitle_best` ambiguity was reduced further without changing
  runtime topology:
  - no new fallback stage was added
  - no new live runtime dependency was added
- two narrow regression tests were added to pin the intended boundary:
  - `pkg/logic/pre_download_process/pre_download_proces_test.go`
    - proves `subtitle_best` does not enter supplier plans without an API key
    - proves `subhd` still stays present when `subtitle_best` supplier role is
      unavailable
  - `pkg/logic/sub_supplier/status_probe_test.go`
    - proves `subtitle_best` reports `credential missing` when explicitly
      enabled without a key instead of pretending to be healthy
- focused verification passed:
  - `go test ./pkg/logic/pre_download_process -count=1`
  - `go test ./pkg/logic/sub_supplier -count=1`
  - `go test ./pkg/subtitle_best_api -count=1`
- practical interpretation:
  - this does not create supplier-proof for `subtitle_best`
  - it does lock in the current intended behavior:
    - supplier role stays out unless both enabled and keyed
    - shared subtitle.best-backed support code may still exist independently

## 2026-06-20 subtitle_best Shared-code Log Noise Downgrade

- one more small runtime polish fix was applied after auditing the retained
  local logs:
  - the current pulled FnOS config intentionally keeps `subtitle_best` supplier
    role disabled with no API key
  - despite that, `PreDownloadProcess.Init()` had been emitting the following
    line as a warning on nearly every startup:
    - `SubtitleBestCodeProvider.GetCode auth key is not set continue without shared code`
  - this was noisy and misleading because it described an expected optional
    downgrade path rather than an operational fault
- implementation:
  - `pkg/logic/pre_download_process/pre_download_proces.go`
  - the `ErrAuthKeyNotSet` branch now logs at `INFO` instead of `WARNING`
  - no route topology was changed
  - no provider enable logic was changed
- focused verification passed:
  - `go test ./pkg/logic/pre_download_process -count=1`
  - `go test ./pkg/logic/sub_supplier -count=1`
- runtime rebuild proof:
  - rebuilt active image:
    - `sha256:638cb652520a5d6c70864a840b1ab1998b8c47ae3139d3aae90cee3866f32dbb`
  - rebuild/start report root:
    - `D:\tmp\csf-local-candidate\reports\20260620-152645-145`
  - live container log now shows:
    - `[INFO]: ... SubtitleBestCodeProvider.GetCode auth key is not set continue without shared code`
  - and no new `[WARNING]` version of that same message appears in the rebuilt
    container startup log
- practical interpretation:
  - this is a signal-quality fix, not a behavior change
  - it reduces false alarm noise around the intentionally disabled
    `subtitle_best` supplier role while preserving the shared-code optional
    downgrade

## 2026-06-20 subtitle_best Default-Order De-drift

- one more strategy drift point was removed from the local candidate code:
  - `pkg/types/common/sub_site_sequence.go`
    - `DefaultPrimarySubSiteSequence()` no longer advertises `subtitle_best` as
      part of the default native-Chinese priority
  - `pkg/types/common/sub_site_sequence_test.go`
    - updated to lock the default primary/shared order to:
      `assrt -> subhd -> shooter -> xunlei -> opensubtitles`
    - also proves omitted suppliers such as `subtitle_best` still remain
      orderable and fall to the tail instead of disappearing
  - `pkg/logic/pre_download_process/pre_download_proces_test.go`
    - updated so primary-plan ordering now proves `subtitle_best` stays behind
      the verified default-primary suppliers when it is present at all
- focused verification passed:
  - `go test ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier -count=1`
- meaning of this change:
  - current runtime behavior under the pulled FnOS config is unchanged, because
    `subtitle_best` supplier role is still disabled without an API key
  - future enabled-with-key behavior is now less misleading because the default
    policy constants no longer overstate `subtitle_best` as a front-of-chain
    default supplier

## 2026-06-20 rebuilt-image smoke after default-order de-drift

- after the code/test de-drift work, the local candidate image was rebuilt again
  so the active runtime matches the current workspace instead of an older image:
  - active image:
    - `sha256:712d52ca807be105c511c7322411d50dd01dae006088bc687e9ad81cb396ea15`
  - active container:
    - `chinesesubfinder-local-candidate`
- one minimal real-library smoke round was then run against the pulled FnOS
  config volumes:
  - round root:
    - `D:\tmp\csf-local-candidate\reports\20260620-154103-049`
  - e2e proof:
    - `D:\tmp\csf-local-candidate\reports\20260620-154252-798-e2e-matrix\e2e-summary.json`
  - requested isolation:
    - primary Chinese suppliers: `__none__`
    - English fallback suppliers: `subdl`
  - result:
    - `route_key=movie.english_fallback`
    - `actual_supplier=subdl`
    - `job_terminal_status=3`
    - `final_output_has_chinese=false`
    - `policy_warnings=[]`
- practical meaning:
  - the rebuilt image still runs correctly against the pulled FnOS config and
    mounted local runtime sample pool
  - the most recent active runtime is no longer the older `638cb652...` image
  - this round added one new retained smoke proof without widening the test
    matrix unnecessarily

## 2026-06-20 additional current-image route proofs on `712d52ca...`

- after the rebuilt-image smoke round, two more minimal live route checks were
  run on the same active image without rebuilding again:
  - image remained:
    - `sha256:712d52ca807be105c511c7322411d50dd01dae006088bc687e9ad81cb396ea15`
- native Chinese primary-chain proof on the current image:
  - `D:\tmp\csf-local-candidate\reports\20260620-154810-112-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese suppliers: `subhd`
    - English fallback suppliers: `subtitlecat`
  - result:
    - `route_key=series.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
    - `final_output_has_chinese=true`
    - `policy_warnings=[]`
- explicit translated-Chinese fallback proof on the current image:
  - `D:\tmp\csf-local-candidate\reports\20260620-154930-032-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese suppliers: `__none__`
    - English fallback suppliers: `subtitlecat`
    - explicit translated fallback enabled
  - result:
    - `route_key=series.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `job_terminal_status=3`
    - `final_output_has_chinese=true`
    - `policy_warnings=[]`
- combined interpretation for the current active runtime:
  - current-image proof set now includes:
    - `movie.english_fallback -> subdl`
    - `movie.native_chinese -> subhd`
    - `movie.safe_fail -> No Sub Found`
    - `series.native_chinese -> subhd`
    - `series.subtitlecat_translated -> subtitlecat_translated`
  - this is still not a full rerun of the whole matrix on `712d52ca...`, but it
    does prove the active runtime is healthy across the three most important
    live stages: native Chinese, English fallback, and explicit translated
    fallback

## 2026-06-20 movie-branch current-image proofs on `712d52ca...`

- the current active image was also given two direct movie-branch checks so the
  refreshed evidence is not overly series-heavy:
  - image remained:
    - `sha256:712d52ca807be105c511c7322411d50dd01dae006088bc687e9ad81cb396ea15`
- movie native-Chinese proof:
  - `D:\tmp\csf-local-candidate\reports\20260620-155530-524-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese suppliers: `subhd`
    - English fallback suppliers: `subtitlecat`
  - result:
    - `route_key=movie.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
    - `final_output_has_chinese=true`
    - `policy_warnings=[]`
- movie safe-fail guard proof:
  - `D:\tmp\csf-local-candidate\reports\20260620-155634-014-e2e-matrix\e2e-summary.json`
  - request shape:
    - primary Chinese suppliers: `__none__`
    - English fallback suppliers: `subtitlecat`
    - `AcceptNoSubFound`
  - result:
    - `route_key=movie.safe_fail`
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
    - `subtitle_file_count=0`
    - `policy_warnings=[]`
- practical meaning:
  - the refreshed current-image evidence now covers both movie success and movie
    guarded-failure paths, not only the fallback-success path

## 2026-06-20 LLM repair-prompt tightening and rebuilt-image verification

- after the current-image LLM checks, one small prompt-only refinement was made
  in `third_party/subflow/src/subflow/translate_job.py`:
  - the repair prompt now explicitly says that very short line-leading English
    dialogue such as `- In secret.` and `- Why secret?` must also be rewritten
    into natural Chinese instead of being copied through
  - a matching unit test was added in:
    - `third_party/subflow/src/subflow/test_translate_job.py`
  - focused verification passed with the project-style `PYTHONPATH` setup:
    - `python -m unittest subflow.test_translate_job`
- the local candidate image was then rebuilt again so the active runtime matched
  that prompt change:
  - active image is now:
    - `sha256:3d7f32f87354d30895845e737afa307ddca4f5850ba27b33c035e48b1c939dab`
- rebuilt-image movie LLM proof:
  - `D:\tmp\csf-local-candidate\reports\20260620-164047-399-e2e-matrix\e2e-summary.json`
  - result:
    - `route_key=movie.llm_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
    - `final_output_has_chinese=true`
- rebuilt-image content audit:
  - latest `movie.llm_fallback` content audit now reports:
    - `dialogue_line_count=2935`
    - `english_only_line_count=4`
    - `english_only_samples=["- NASA？","- NASA。","1G。","RPM。"]`
  - the earlier clearly untranslated short dialogue residues
    `- In secret.` and `- Why secret?` no longer appear in the English-only
    sample set after the rebuilt-image rerun
- practical interpretation:
  - the latest prompt tightening improved the exact short-fragment failure mode
    that was still visible in the previous movie LLM proof
  - remaining English-only residues on the rebuilt image are now limited to
    allowed acronym / technical-symbol style cases rather than missed ordinary
    dialogue translation

## 2026-06-20 same-image rerun after acronym-only LLM tightening

- one more prompt-only tightening plus cue-kind guard was added in:
  - `third_party/subflow/src/subflow/translate_job.py`
  - `third_party/subflow/src/subflow/test_translate_job.py`
- focused verification passed again:
  - `python -m unittest subflow.test_translate_job`
  - `go test ./pkg/downloader ./pkg/logic/mark_system ./pkg/types/common ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier -count=1`
- a new local candidate image was then rebuilt and became active:
  - active image:
    - `sha256:dc20fe0945ce...`
- a full same-image acceptance attempt was started with DeepSeek-backed LLM:
  - command path:
    - `scripts/local_full_acceptance.ps1`
  - result:
    - progressed through `movie.native_chinese` and
      `movie.subtitlecat_translated`
    - stopped at `movie.english_fallback`
    - hard failure reason from harness:
      `Requested supplier prerequisites are missing in the pulled FnOS config: subdl api key`
- this failure is configuration-boundary evidence, not a route-regression proof:
  - current pulled FnOS config does not currently permit a true same-image
    re-proof of the default `subdl -> ...` English fallback route
- same-image reruns that did complete on `dc20fe0945ce...`:
  - `D:\tmp\csf-local-candidate\reports\20260620-173602-799-e2e-matrix\e2e-summary.json`
    - `route_key=movie.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
  - `D:\tmp\csf-local-candidate\reports\20260620-173709-104-e2e-matrix\e2e-summary.json`
    - `route_key=movie.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `job_terminal_status=3`
  - `D:\tmp\csf-local-candidate\reports\20260620-180330-023-e2e-matrix\e2e-summary.json`
    - `route_key=series.native_chinese`
    - `actual_supplier=subhd`
    - `job_terminal_status=3`
  - `D:\tmp\csf-local-candidate\reports\20260620-180702-159-e2e-matrix\e2e-summary.json`
    - `route_key=series.subtitlecat_translated`
    - `actual_supplier=subtitlecat_translated`
    - `job_terminal_status=3`
  - `D:\tmp\csf-local-candidate\reports\20260620-181014-844-e2e-matrix\e2e-summary.json`
    - `route_key=movie.safe_fail`
    - `job_terminal_status=2`
    - `job_error_info=No Sub Found`
- newest movie LLM fallback rerun on the same active image:
  - `D:\tmp\csf-local-candidate\reports\20260620-174022-018-e2e-matrix\e2e-summary.json`
    - `route_key=movie.llm_fallback`
    - `actual_supplier=subtitlecat`
    - `job_terminal_status=3`
    - `final_output_has_chinese=true`
- newest content audit:
  - `powershell -ExecutionPolicy Bypass -File scripts/local_subtitle_content_audit.ps1 -RouteKeys movie.llm_fallback -AsJson`
  - result:
    - `dialogue_line_count=2940`
    - `english_only_line_count=0`
    - mixed-language lines remain only as embedded accepted tokens such as
      `NASA`, `GPS`, `MRI`, `A计划`, `B计划`
- operational cleanup:
  - sequential residue audit + cleanup was rerun after these rounds
  - latest retained residue audit:
    - `D:\tmp\csf-local-candidate\reports\residue-audit-20260620-181153-146.json`
- one additional harness footnote surfaced:
  - the newest `series.native_chinese` and `series.subtitlecat_translated`
    `e2e-summary.json` files contain malformed JSON escaping in sample-name
    fields, but the route assertions, supplier assertions, terminal status, and
    subtitle outputs are all present in-file and were extracted directly for
    evidence

## 2026-06-20 latest-image movie English-fallback route completion

- the newest active image `dc20fe0945ce...` was also used to complete an
  explicit movie English-fallback proof with the actually available pulled-config
  supplier set:
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
- practical meaning:
  - the latest-image route-evidence set now contains all requested route
    categories:
    - `movie.native_chinese`
    - `series.native_chinese`
    - `movie.english_fallback`
    - `series.subtitlecat_translated`
    - `movie.safe_fail`
  - the narrower outstanding configuration boundary is no longer route coverage
    itself, but only the absence of a `subdl` credential in the pulled FnOS
    config for re-proving the exact default `subdl`-led English supplier order
