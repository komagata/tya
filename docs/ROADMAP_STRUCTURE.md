# Roadmap Structure

Roadmap Structure is a compact text format for showing roadmap hierarchy and
progress in a way that remains readable in both monospace and proportional
font environments.

The format has one root Goal. Under the Goal are one or more Epics. Under each
Epic are one or more Milestones. Under each Milestone are one or more Tasks.

```text
Goal
    Epic
        Milestone
            Task
```

Each item is written on one line.

```text
{progress_bar} {percent}% {number} {title}
```

The Goal line has no number because there is only one Goal.

```text
{progress_bar} {percent}% {goal_title}
```

## Example

```text
Roadmap Structure

████░░░░░░ 45% Self-host tya
    ████░░░░░░ 40% 1 Self-host AST migration
        ████░░░░░░ 40% 1-1 Parser AST化
            ████████░░ 80% 1-1-1 expression AST対応
            ░░░░░░░░░░ 0% 1-1-2 statement AST対応
        ██░░░░░░░░ 20% 1-2 Checker対応
            ░░░░░░░░░░ 0% 1-2-1 AST node type checking
        █░░░░░░░░░ 10% 1-3 Codegen対応
            █░░░░░░░░░ 10% 1-3-1 AST code generation
    ██████░░░░ 60% 2 CLI usability
        ███████░░░ 70% 2-1 Error output
            ██████████ 100% 2-1-1 行番号と列番号を表示
            ████░░░░░░ 40% 2-1-2 diagnostic message整理
        ███░░░░░░░ 30% 2-2 Commands
            ░░░░░░░░░░ 0% 2-2-1 format command追加
            ░░░░░░░░░░ 0% 2-2-2 inspect AST command追加
    ███░░░░░░░ 30% 3 Documentation
        ██████████ 100% 3-1 Roadmap
            ██████████ 100% 3-1-1 Roadmap Structure定義
        ██░░░░░░░░ 20% 3-2 Self-host
            ██░░░░░░░░ 20% 3-2-1 AST移行手順を書く
        ███░░░░░░░ 30% 3-3 CLI
            ███░░░░░░░ 30% 3-3-1 usage examplesを整理
```

## Terms

- Goal: the single top-level objective of the roadmap.
- Epic: a large outcome that contributes to the Goal.
- Milestone: a meaningful checkpoint that completes part of an Epic.
- Task: a leaf-level work item with implementation, docs, and verification.

## Numbering

- Epics use integer numbers: `1`, `2`, `3`.
- Milestones use the Epic number plus a local number: `1-1`, `1-2`, `2-1`.
- Tasks use the Milestone number plus a local number: `1-1-1`, `1-1-2`,
  `2-1-1`.
- The number describes the item's position in the hierarchy.
- Renumber items when the roadmap is reorganized.

## Progress Bars

- Use a 10-cell progress bar.
- Use `█` for completed cells.
- Use `░` for incomplete cells.
- Put the percentage to the right of the bar.
- Do not put variable-width text to the left of the bar.
- Round progress to the nearest 10% for the bar. Keep the numeric percentage
  accurate enough for planning.

Examples:

```text
░░░░░░░░░░ 0%
█░░░░░░░░░ 10%
█████░░░░░ 50%
██████████ 100%
```

## Indentation

- The Goal is not indented.
- Epics are indented by 4 spaces.
- Milestones are indented by 8 spaces.
- Tasks are indented by 12 spaces.
- Keep exactly one item per line.
- Do not add blank lines between Epics.

## Completion Rules

- A Task is complete when its implementation, docs, and required verification
  are complete.
- A Milestone is complete when every Task below it is complete.
- An Epic is complete when every Milestone below it is complete.
- The Goal progress is derived from the Epics.

## Maintenance Rules

- Keep active work near the top.
- Do not update `ROADMAP.md` for every small implementation slice.
- Report small completed slices in chat using this format instead of changing
  the roadmap immediately.
- Batch `ROADMAP.md` updates after several related Tasks complete, when a
  Milestone changes meaningfully, or when the strategy changes.
- Treat `ROADMAP.md` as the stable remaining-work plan, not as a chronological
  progress log.
- When finishing work, remove the completed Task from the roadmap instead of
  keeping historical completion records there.
- If all Tasks under a Milestone have been removed, remove the Milestone too
  unless it still has meaningful remaining work.
- If all Milestones under an Epic have been removed, remove the Epic too unless
  it still has meaningful remaining work.
- If a supporting document contains a task list, replace it with a pointer to
  the relevant roadmap task.
- Update `ROADMAP.md` only in the same change that meaningfully changes the
  remaining-work plan.

## Stability Rules

`ROADMAP.md` is a planning document, not a work diary. To prevent roadmap churn,
apply these rules before editing it:

- Do not append a new status bullet for each passing test, helper extraction,
  fixture adjustment, or narrow implementation slice.
- Do not rewrite task titles only to match the latest implementation wording if
  the remaining outcome is unchanged.
- Do not renumber items unless the hierarchy itself changes.
- Keep completed micro-work out of `ROADMAP.md`; report it in chat using the
  Roadmap Structure format instead.
- Batch roadmap edits into one update after several related slices complete, a
  Task is fully removed, a Milestone changes scope, or the strategy changes.
- If progress needs to be recorded between roadmap updates, use a short chat
  report or a dedicated design/status note. Do not grow `ROADMAP.md` as a
  chronological log.
- Before editing `ROADMAP.md`, state which of these edit reasons applies:
  completed Task removal, Milestone scope change, Epic scope change, strategy
  change, or verification policy change.
- If none of those reasons applies, leave `ROADMAP.md` unchanged.
- Do not treat a passing focused test, a newly covered example, or a helper
  extraction as a roadmap edit reason by itself.
- Prefer updating chat progress after every small Task and updating
  `ROADMAP.md` only after a coherent batch changes the remaining-work plan.

## Readability Test

When changing this format, test it with:

- one Goal.
- multiple Epics.
- multiple Milestones under each Epic.
- multiple Tasks under each Milestone.
- one item per line.
- percentage text on the right side of the progress bar.
- no variable-width text on the left side of the progress bar.
- no blank lines between Epics.
- readable output in both monospace and proportional font environments.
