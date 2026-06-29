# LLM Subtitle Translation Optimization Plan

## Objective

Improve the general quality and stability of the LLM English-to-Chinese subtitle fallback without overfitting to one episode, one provider, or one franchise.

This plan targets the failure patterns observed in recent real outputs:

- cue-to-cue semantic drift
- unstable handling of short fragmented dialogue
- mixed Chinese/English name leakage
- weak handling of italic/system/broadcast cues
- awkward punctuation and line-break artifacts

The goal is not to force perfect literary translation. The goal is to make fallback subtitles consistently usable, readable, and structurally reliable across typical film and TV subtitles.

## Non-goals

- Do not build title-specific dictionaries for one show.
- Do not hardcode franchise-specific names or slang.
- Do not add a heavy multi-stage fallback stack.
- Do not turn the prompt into a brittle checklist with hundreds of narrow rules.
- Do not add expensive full-episode second-pass translation by default.

## Core diagnosis

The current translation chain is already good at:

- producing JSON-shaped output
- keeping cue count mostly intact
- repairing obvious untranslated English leftovers

The current chain is weak at:

1. preserving strict cue boundaries when neighboring cues are short and dense
2. keeping recurring spoken names in stable Chinese form
3. handling italic/system/broadcast text with a coherent subtitle style
4. cleaning up awkward punctuation and broken line layout after translation
5. distinguishing between:
   - meaningful short dialogue
   - context-only neighboring cues
   - raw labels / watermark / release noise

## Design principles

1. Keep rules general:
   Rules must describe stable subtitle behavior, not one episode's wording.

2. Prefer boundary safety over aggressive rewriting:
   When a cue is ambiguous, it is better to stay slightly literal than to borrow meaning from a neighboring cue and shift content into the wrong slot.

3. Use context as support, not as translation payload:
   Neighboring cues should help disambiguate fragments, but should not be translated into the current cue.

4. Separate semantic quality from cleanup quality:
   Translation prompt should handle meaning.
   Post-processing should handle punctuation, names, and line presentation.

5. Keep cost growth limited:
   Default flow should remain single-pass plus lightweight repair/cleanup, not become a deep multi-pass pipeline.

## Execution plan

### Phase 1: Tighten the main translation contract

#### Change

Refine the main prompt in `third_party/subflow/src/subflow/translate_job.py` so that it more clearly enforces cue boundary discipline.

#### Required prompt adjustments

- Explicitly state that each cue may use neighboring cues only for local disambiguation.
- Explicitly forbid moving substantive meaning from the previous or next cue into the current cue.
- Explicitly prefer conservative translation when a fragment is unclear.
- Explicitly require stable Chinese rendering for recurring spoken person/place/object names when local context is sufficient.
- Explicitly distinguish spoken dialogue from system/broadcast/on-screen cues, but keep the rule general.

#### Constraints

- Do not add long enumerations of franchise-specific examples.
- Keep the prompt readable and provider-agnostic.
- Avoid rule duplication between main prompt and repair prompt.

#### Acceptance criteria

- Short fragmented dialogue no longer frequently absorbs content from neighboring cues.
- Prompt text remains compact enough to avoid large token overhead.

### Phase 2: Switch from pure chunk translation to target-plus-context translation

#### Change

Adjust chunk construction so the model sees:

- a target cue set to translate
- a small context window before and after

but is allowed to return translations only for target cues.

#### Why

Current chunking gives local context, but all cues in the chunk are translation targets. That encourages semantic bleeding between short adjacent cues.

#### Proposed approach

- Keep target batch size moderate.
- Attach a small number of surrounding cues as `context_only`.
- Prompt the model that context-only cues exist for interpretation, not for output.

#### Initial scope

- Keep default target chunk size close to the current scale.
- Start with a small context window, not a large one.
- Reuse the same JSON response format.

#### Constraints

- Do not double or triple total token usage aggressively.
- Do not introduce full-episode memory or episode summaries in the default path.

#### Acceptance criteria

- Fewer cue drift cases on dense short-dialogue sequences.
- No increase in cue drop or cue count mismatch.

### Phase 3: Upgrade repair from "English leftover repair" to "boundary-safe cleanup repair"

#### Change

Keep the current repair stage lightweight, but expand what it can fix.

#### Repair should cover

- untranslated or half-translated English fragments
- recurring bare English names that should become stable Chinese renderings
- mixed Chinese/English label leakage
- suspicious ultra-short unnatural outputs that look like borrowed neighboring content

#### Repair should not do

- free rewrite of already acceptable subtitle lines
- full-scene stylistic polishing
- title-specific vocabulary injection

#### Constraints

- Repair remains selective and conditional.
- Trigger repair only for suspicious cues, not whole chunks by default.

#### Acceptance criteria

- Fewer mixed-script outputs like `Summer`, `Rick`, `A+`, `GTA` surviving in spoken dialogue when a normal Chinese rendering is expected.
- Repair does not noticeably increase hallucinated rewrites.

### Phase 4: Add a dedicated post-processing cleanup pass

#### Change

Add a deterministic cleanup layer after translation and repair.

#### Cleanup responsibilities

- normalize punctuation spacing
- remove isolated punctuation-only lines
- fix malformed line breaks like trailing slash + punctuation artifacts
- normalize repeated Chinese/English mixed name forms where confidence is high
- preserve markup safely while cleaning visible text

#### Important rule

This pass should be formatting-aware, not semantics-heavy.

It may normalize obvious presentation artifacts, but must not invent missing meaning.

#### Constraints

- Keep the cleanup deterministic.
- Prefer whitelist-like structural fixes over semantic guesswork.

#### Acceptance criteria

- Obvious artifacts such as ` / 。` disappear.
- Subtitle lines look closer to deliverable Chinese subtitle layout.

### Phase 5: Improve cue classification without over-specialization

#### Change

Strengthen generic cue classification used by prompt payload generation.

#### Target categories

- dialogue
- italic dialogue / lyric / off-screen speech
- system / alert / broadcast / on-screen informational text
- irrelevant release / watermark / file-noise text

#### Why

Several bad outputs come from treating all short italicized lines as the same kind of content.

#### Constraints

- Use structural cues already present in subtitle text.
- Do not add show-specific lexical rules.

#### Acceptance criteria

- System and broadcast lines become more consistent in tone.
- Irrelevant noise remains filtered without suppressing meaningful on-screen text.

### Phase 6: Build a focused regression set from real failure shapes

#### Change

Create a small regression corpus from real-world failure patterns, not from one title only.

#### Corpus should include

- dense short dialogue exchanges
- alternating multi-speaker fragments
- italic/system/broadcast cues
- recurring proper names
- acronym and short-token lines
- punctuation / line-break edge cases

#### Source policy

- Use small anonymized or local test snippets.
- Cover multiple titles and styles.

#### Acceptance criteria

- A prompt or chunking change cannot be accepted unless it improves or preserves this regression set.

### Phase 7: Validate with quality gates

#### Required checks

1. Structural checks
   - cue count stability
   - no malformed JSON
   - no unsupported line explosion

2. Heuristic checks
   - reduced ASCII leakage in spoken dialogue
   - reduced punctuation artifacts
   - reduced suspicious cue drift markers

3. Human spot checks
   - at least one dense dialogue sequence
   - at least one system/italic sequence
   - at least one name-heavy sequence

#### Acceptance criteria

- Changes improve readability without significantly increasing mistranslation rate.
- Token growth stays moderate.

## Recommended implementation order

1. Phase 1: tighten main prompt
2. Phase 2: target-plus-context chunking
3. Phase 4: deterministic cleanup pass
4. Phase 3: expand selective repair triggers
5. Phase 5: cue classification refinement
6. Phase 6 and 7: regression and validation

This order is intentional:

- prompt and chunking fix the biggest semantic issue first
- cleanup removes visible ugliness next
- repair expansion comes after boundary safety is improved

## File touch map

Expected primary files:

- `third_party/subflow/src/subflow/translate_job.py`
- `third_party/subflow/src/subflow/test_translate_job.py`
- `pkg/llm_subtitle_fallback/manager.go`
- optional new local regression fixtures and tests under the existing subtitle fallback test area

## Success definition

The optimization is successful when fallback subtitles become:

- structurally reliable
- semantically less drift-prone
- visually cleaner
- more consistent on names and short cues

without turning the pipeline into an over-engineered, high-cost, multi-pass system.
