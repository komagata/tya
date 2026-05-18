---
name: spec-questioner
description: Ask the user focused specification questions with recommended choices, then write an implementation-ready feature spec that includes acceptance criteria and concrete tests to add. Use when the user asks to clarify a language/product design, find remaining decisions, turn decisions into a SPEC, or says phrases like "他にも明確化したほうがいい点があれば僕に質問して、その結果をテストすることを含めた SPEC として作成して".
---

# Spec Questioner

## Overview

Use this skill to drive a decision loop: ask only non-obvious, high-impact specification questions, recommend a default for each, collect the user's answers, and write a testable feature spec. Do not implement code or tests unless the user explicitly asks to switch from spec work to implementation.

When the user invokes `$spec-questioner` without additional detail, treat it as this full request:

```text
他にも明確化したほうがいい点があれば僕に質問して、その結果をテストすることを含めた SPEC として作成して
```

## Workflow

1. Inspect nearby existing specs and docs before asking.
   - In this repo, read relevant files under `feature-specs/`, `docs/SPEC.md`, `docs/STRICT_SEMANTICS.md`, and nearby tests only as needed.
   - Avoid repeating questions already decided in existing specs unless there is a conflict.

2. Decide what not to ask.
   - Do not ask about behavior that is obvious from the existing code, existing docs, common CLI/language convention, or prior user decisions.
   - Do not ask for confirmation when there is one clearly dominant choice and the risk is low. Record that choice as an assumption in the eventual spec instead.
   - Ask only when a reasonable implementation could go more than one way and the choice affects user-visible behavior, compatibility, tests, or long-term language design.
   - If a decision is self-evident but important, include it later in the spec's Behavior or Context without interrupting the user.

3. Ask a compact batch of decision questions.
   - Include an explicit recommendation for every question.
   - Prefer 5-10 questions. Use more only when every item is genuinely non-obvious.
   - For each question, state the behavioral tradeoff in concrete terms.
   - Keep questions answerable with "おすすめ", "OK", "禁止", "A/B", or short free text.

4. Explain any unclear choice before writing a spec.
   - If the user asks what an option means, answer that option only.
   - Include pros/cons and a recommended choice.
   - Resume the decision loop after the user answers.

5. Write one feature spec after decisions are resolved.
   - Use `feature-specs/<short-slug>.md` in this repo.
   - The file's presence under `feature-specs/` means it is implementation-ready.
   - Do not create draft files, status frontmatter, or unresolved placeholders.
   - If the decisions are too broad for one coherent spec, split by theme and create one spec for the current theme.

6. Verify the spec artifact only.
   - Check the new spec has no unresolved markers such as `TODO`, `TBD`, `unresolved`, `??`, `未定`, or `決め`.
   - Do not run implementation tests for spec-only work unless the user asks.

## Question Style

Before asking, silently filter out self-evident items. The final question list should feel like "these are the few choices where user judgment matters", not an exhaustive checklist.

Use this shape:

```text
1. **Topic**
   おすすめ: <recommended behavior>.
   <short reason/tradeoff>.
   これでよいですか？
```

When comparing choices:

```text
7. **Topic**
   A. <choice>
   メリット: ...
   デメリット: ...

   B. <choice>
   メリット: ...
   デメリット: ...

   おすすめ: A
```

## Spec Format

Use this structure unless a stronger local template exists:

```md
# Feature: <Name>

## Goal

<Short user-visible result.>

## Context

<Current behavior, related specs, and constraints.>

## Behavior

- <Concrete accepted behavior.>

## Scope

- <Files, modules, docs, tests, or commands expected to change during implementation.>

## Out of Scope

- <Explicit exclusions.>

## Acceptance Criteria

- <Observable pass/fail criteria.>

## Tests To Add

Parser/checker tests:

- `<TestName>`
  - <Snippet and expected result.>

Eval tests:

- `<TestName>`
  - <Snippet and expected result.>

Testscript coverage:

- <End-to-end fixture expectations.>

## Verification

```sh
<focused commands>
```
```

## Quality Bar

A completed spec must:

- record only decisions the user accepted;
- record low-risk self-evident assumptions without pretending they were user decisions;
- distinguish current implementation from future implementation if they differ;
- include edge cases and error behavior;
- include concrete tests to add, not only prose;
- include verification commands;
- preserve repository constraints such as self-host compatibility when relevant;
- contain no unresolved questions or placeholders.

## Guardrails

- Do not quietly choose behavior for a meaningful open design decision. Ask.
- Do not ask the user to approve obvious conventions or low-risk implementation details.
- Do not add implementation changes while using this skill.
- Do not add roadmap entries unless the user asks.
- Do not present a "future idea" as accepted current behavior. Use clear wording such as "implementation-ready spec" or "accepted behavior for a future implementation pass".
- Prefer the user's "迷わない" principle: one canonical spelling, deterministic behavior, and explicit errors over silent fallback.
