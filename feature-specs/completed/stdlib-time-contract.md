# Feature: stdlib Time Contract

## Goal

Freeze a practical v1.0.0 time API for timestamps, durations, UTC/local
conversion, formatting, parsing, arithmetic, and monotonic elapsed time.

## Context

Tya currently exposes `time/Time` with `now`, `sleep`, `format`, `parse`, and
`since`. That is enough for simple programs, but v1.0.0 needs a clearer public
contract for the common time operations expected by CLI tools, servers, tests,
and logs.

The accepted direction is to keep one `time/Time` class-style package and avoid
adding date/time syntax. Time values remain runtime values backed by the
compiled runtime.

## Behavior

- `time.Time.now()` returns the current wall-clock time value.
- `time.Time.monotonic()` returns a monotonic timestamp value suitable for
  elapsed-time measurement.
  - Monotonic values are not formatted as wall-clock dates.
  - Subtracting or comparing monotonic values is valid.
- `time.Time.unix(seconds, nanos = 0)` constructs a UTC time from Unix seconds
  and nanoseconds.
- Wall-clock time values support:
  - `unix()`: seconds since Unix epoch;
  - `unix_nanos()`: nanoseconds since Unix epoch;
  - `utc()`: equivalent UTC time value;
  - `local()`: equivalent local time value;
  - `format(layout)`: string;
  - `add(duration)`: time value;
  - `sub(other)`: duration when `other` is a time value.
- `time.Time.parse(text, layout = "rfc3339")` parses a time string.
  - Supported layout names include `rfc3339`, `date`, and `unix`.
  - Invalid text raises a structured time error.
- `time.Time.duration(seconds = 0, options = {})` constructs a duration value.
  - Supported options: `minutes`, `hours`, `milliseconds`, `microseconds`,
    `nanoseconds`.
  - Unknown option keys are invalid.
- Duration values support:
  - `seconds()`, `milliseconds()`, `microseconds()`, `nanoseconds()`;
  - `add(other)`, `sub(other)`;
  - comparison through numeric duration value semantics or explicit methods;
  - display as a deterministic human-readable string.
- `time.Time.sleep(duration_or_seconds)` accepts either a duration value or a
  number of seconds.
- Timezone support for v1.0.0 is limited to UTC and the process local timezone.
  - Named timezone database lookup is not part of v1.0.0.
  - Formatting may include UTC offset when supported by the layout.
- All time failures raise structured errors with `kind: "time"` and stable
  `code` values.

## Scope

- `lib/time/Time.tya`
- runtime-backed time and duration values for interpreter and generated C
- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- runtime/codegen parity tests
- release-platform behavior for Linux, macOS, and Windows

## Out of Scope

- Date/time literal syntax.
- Named timezone database support.
- Locale-aware month/day names.
- Calendar recurrence APIs.
- Leap-second guarantees beyond host runtime behavior.
- Chronology systems other than the Unix/Gregorian baseline.

## Acceptance Criteria

- `docs/SPEC.md` documents the v1 time and duration API.
- Time parsing and formatting are deterministic for the documented layouts.
- UTC and local conversions work on supported release platforms.
- Duration construction and arithmetic work in interpreter and generated C.
- Monotonic elapsed-time measurement is distinct from wall-clock formatting.
- Invalid layouts, invalid parse text, wrong argument kinds, and unknown
  duration options raise structured errors with stable codes.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Eval/runtime tests:

- `TestRunTimeUnixFormatParse`
  - Constructs a known Unix timestamp, formats it as RFC3339, parses it back,
    and checks Unix seconds.

- `TestRunTimeDurationArithmetic`
  - Constructs durations from seconds and option units.
  - Expected: add/sub and unit conversion methods return deterministic values.

- `TestRunTimeUtcLocalBoundaries`
  - Converts a known time to UTC and local.
  - Expected: UTC output is deterministic; local conversion preserves instant.

- `TestRunTimeMonotonicElapsed`
  - Captures monotonic values around a short sleep.
  - Expected: elapsed duration is non-negative and not formatted as wall-clock
    time.

- `TestRunTimeStructuredErrors`
  - Invalid parse input, invalid layout, unknown option, and wrong kinds.
  - Expected: structured time errors with stable codes.

Codegen tests:

- `TestEmitCTimeContractProgram`
  - Builds and runs timestamp, duration, parse, format, and sleep usage.

Testscript coverage:

- `v1_stdlib_time_contract.txtar`
  - Covers CLI-level valid and invalid time API behavior.

Documentation tests:

- `TestSpecDocumentsTimeContract`
  - Expected: `docs/SPEC.md` documents wall-clock time, monotonic time,
    durations, UTC/local scope, and unsupported named timezones.

## Verification

```sh
go test ./internal/eval -run Time -count=1
go test ./internal/codegen -run Time -count=1
go test ./tests -run 'TestV.*Scripts|TestSpecDocumentsTimeContract|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
