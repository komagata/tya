# Goal Tree

Goal Tree is a compact format for describing a hierarchical, status-tracked
plan as a rooted tree. It is readable in both monospace and proportional font
environments and has three official renderings (plain text, Markdown, HTML)
that all encode the same model.

The name "Goal Tree" refers to both the abstract model and the format. A
project's `ROADMAP.md` is one application of Goal Tree, but the format itself
is not tied to roadmaps and can describe any goal-rooted plan.

## Model

The format defines an abstract model first, then a textual rendering of that
model. Both must agree.

A roadmap is a **tree with exactly one root node**. The root is the Goal.

Every node in the tree has:

- a **kind**, which is one of `Goal`, `Epic`, `Milestone`, or `Task`;
- a **title**, a non-empty string;
- a **status**, one of `complete` or `incomplete`, expressible as a checkbox.

The tree has exactly **four levels of kinds**, in this fixed parent-child order:

```text
Goal
  Epic
    Milestone
      Task
```

- The Goal is the unique root.
- An Epic's parent must be the Goal.
- A Milestone's parent must be an Epic.
- A Task's parent must be a Milestone.
- A node's children must all be of the next kind below it; mixing kinds at the
  same level is not allowed.
- A Task is a leaf and has no children.

Status derivation:

- A Task's status is set directly.
- A non-leaf node (Goal, Epic, Milestone) is `complete` if and only if every
  child is `complete`. Otherwise it is `incomplete`.

Everything else in this document — progress bars, percentages, hierarchical
numbering, indentation widths — is a rendering of this model, not part of the
model itself.

## Renderings

The model defined above can be written out in three official renderings:
**plain text**, **Markdown**, and **HTML**. All three encode the same tree, the
same kinds, the same titles, and the same statuses; converting between them
must be lossless for the model.

### Plain text rendering

Plain text uses indentation, a 10-cell progress bar, a percentage, an optional
hierarchical number, and the title, one node per line.

```text
Goal
    Epic
        Milestone
            Task
```

Per-line shape:

```text
{progress_bar} {percent}% {number} {title}
```

The Goal has no number:

```text
{progress_bar} {percent}% {goal_title}
```

The progress bar and percent are derived from the model's status: a `complete`
node renders as `██████████ 100%`, an `incomplete` leaf renders as
`░░░░░░░░░░ 0%`, and an `incomplete` non-leaf renders the share of complete
descendants as defined in *Progress Bars* below. See *Indentation* and
*Numbering* for the remaining details.

Plain text example:

```text
Goal Tree

████░░░░░░ 45% Self-host tya
    ████░░░░░░ 40% 1 Self-host AST migration
        ████░░░░░░ 40% 1-1 Parser AST migration
            ████████░░ 80% 1-1-1 Expression AST support
            ░░░░░░░░░░ 0% 1-1-2 Statement AST support
        ██░░░░░░░░ 20% 1-2 Checker support
            ░░░░░░░░░░ 0% 1-2-1 AST node type checking
        █░░░░░░░░░ 10% 1-3 Codegen support
            █░░░░░░░░░ 10% 1-3-1 AST code generation
    ██████░░░░ 60% 2 CLI usability
        ███████░░░ 70% 2-1 Error output
            ██████████ 100% 2-1-1 Show line and column numbers
            ████░░░░░░ 40% 2-1-2 Organize diagnostic messages
        ███░░░░░░░ 30% 2-2 Commands
            ░░░░░░░░░░ 0% 2-2-1 Add format command
            ░░░░░░░░░░ 0% 2-2-2 Add inspect AST command
    ███░░░░░░░ 30% 3 Documentation
        ██████████ 100% 3-1 Roadmap
            ██████████ 100% 3-1-1 Define Goal Tree
        ██░░░░░░░░ 20% 3-2 Self-host
            ██░░░░░░░░ 20% 3-2-1 Write AST migration steps
        ███░░░░░░░ 30% 3-3 CLI
            ███░░░░░░░ 30% 3-3-1 Organize usage examples
```

### Markdown rendering

Markdown uses a top-level heading for the Goal and a GitHub Flavored Markdown
nested task list for Epics, Milestones, and Tasks. The status of each Epic /
Milestone / Task maps to the GFM checkbox: `- [x]` for `complete`, `- [ ]` for
`incomplete`. The Goal has no checkbox; its status is derived from its Epics.

Per-node shape:

- Goal: `# {title}`
- Epic: `- [ ] {title}` or `- [x] {title}` at the top level of the list
- Milestone: indented two spaces under its Epic, same checkbox shape
- Task: indented four spaces under its Milestone, same checkbox shape

Hierarchical numbers and progress bars are not part of the Markdown rendering;
GitHub displays an automatic completion count for each list. The Markdown
rendering is the canonical form for `ROADMAP.md` in this repository.

Markdown example:

```markdown
# Self-host tya

- [ ] Self-host AST migration
  - [ ] Parser AST migration
    - [ ] Expression AST support
    - [ ] Statement AST support
  - [ ] Checker support
    - [ ] AST node type checking
- [x] Documentation
  - [x] Roadmap
    - [x] Define Goal Tree
```

### HTML rendering

HTML uses semantic list elements with `data-` attributes that carry the model
fields, so it can be rendered with progress bars and numbers in a browser while
remaining machine-readable.

- Goal: `<h1 class="roadmap-goal" data-status="...">{title}</h1>`
- The list of Epics is wrapped in `<ul class="roadmap">`.
- Each non-Goal node is `<li data-kind="epic|milestone|task" data-status="complete|incomplete" data-number="1-2-3">{title}</li>`.
- A node with children contains a nested `<ul>` after its title.
- Progress bars and percentages, when shown, are emitted as additional inline
  elements (e.g. `<span class="roadmap-bar">██████████</span>`) and are derived
  from `data-status`, not authored separately.

HTML example:

```html
<h1 class="roadmap-goal" data-status="incomplete">Self-host tya</h1>
<ul class="roadmap">
  <li data-kind="epic" data-status="incomplete" data-number="1">Self-host AST migration
    <ul>
      <li data-kind="milestone" data-status="incomplete" data-number="1-1">Parser AST migration
        <ul>
          <li data-kind="task" data-status="incomplete" data-number="1-1-1">Expression AST support</li>
          <li data-kind="task" data-status="incomplete" data-number="1-1-2">Statement AST support</li>
        </ul>
      </li>
    </ul>
  </li>
  <li data-kind="epic" data-status="complete" data-number="2">Documentation
    <ul>
      <li data-kind="milestone" data-status="complete" data-number="2-1">Roadmap
        <ul>
          <li data-kind="task" data-status="complete" data-number="2-1-1">Define Goal Tree</li>
        </ul>
      </li>
    </ul>
  </li>
</ul>
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
  Goal Tree format instead.
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
