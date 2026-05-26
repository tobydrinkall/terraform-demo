# Decision 3.1 — Conditional Attribute Requirements for `devin_schedule`

## Context

The `devin_schedule` resource has conditional attribute requirements:

| `schedule_type` | Required attribute | Forbidden attribute |
|---|---|---|
| `recurring` | `frequency` (cron expression) | `scheduled_at` |
| `one_time` | `scheduled_at` (ISO 8601 datetime) | `frequency` |

Terraform's schema does not natively support "required-if" semantics. This report evaluates three implementation approaches.

**PoC location:** `internal/provider/schedule_poc/`
**Example configs:** `examples/schedules_poc/`

---

## 1. Error Message Comparison (Plan-Time vs Apply-Time)

### Option A — CRUD-time validation

Errors appear **at apply time only**. The plan succeeds even for invalid configurations.

```
$ terraform apply

Error: Missing required attribute for recurring schedule

  When schedule_type is "recurring", the "frequency" attribute must be set
  to a valid cron expression. Got schedule_type="recurring" with no
  frequency value.

Error: Invalid attribute for recurring schedule

  The "scheduled_at" attribute must not be set when schedule_type is
  "recurring". Got scheduled_at="2025-06-01T09:00:00Z".
```

**Timing:** User writes config → `terraform plan` succeeds ✓ → `terraform apply` fails ✗
**Impact:** The user believes their config is valid after plan, only to be surprised at apply.

### Option B — SDK validators (ConflictsWith + CustomizeDiff)

All errors appear **at plan time**. Three layers of validation:

1. **ConflictsWith** — prevents both `frequency` and `scheduled_at` from being set:
   ```
   Error: Conflicting configuration arguments

     "frequency": conflicts with scheduled_at
   ```

2. **CustomizeDiff** — enforces conditional requirement:
   ```
   Error: "frequency" is required when schedule_type is "recurring"
   ```

3. **ValidateFunc** — catches format errors:
   ```
   Error: expected value of frequency to match regular expression
     "^(\S+\s+){4}\S+$"

     must be a valid 5-field cron expression (e.g. '0 9 * * 1-5')
   ```

**Timing:** User writes config → `terraform plan` fails immediately ✗ → user corrects
**Impact:** Fast feedback loop. No wasted API calls.

### Option C — Two separate resources

All errors appear **at plan time**, enforced by Terraform's own schema parser:

1. **Missing required field:**
   ```
   Error: Missing required argument

     The argument "frequency" is required, but no definition was found.
   ```

2. **Unknown attribute (user tries to set `scheduled_at` on `devin_recurring_schedule`):**
   ```
   Error: Unsupported argument

     An argument named "scheduled_at" is not expected here.
   ```

3. **ValidateFunc** catches format errors (same as Option B).

**Timing:** User writes config → `terraform plan` fails at parse/validate → user corrects
**Impact:** Fastest feedback. The wrong attribute literally does not exist in the schema.

### Summary Table

| Scenario | Option A | Option B | Option C |
|---|---|---|---|
| Missing required attr | Apply-time error | Plan-time (CustomizeDiff) | Plan-time (schema Required) |
| Wrong attr set | Apply-time error | Plan-time (ConflictsWith) | Plan-time (Unsupported argument) |
| Both attrs set | Apply-time error | Plan-time (ConflictsWith) | Impossible (separate schemas) |
| Invalid format | Apply-time error | Plan-time (ValidateFunc) | Plan-time (ValidateFunc) |
| `terraform plan` passes for invalid config? | **Yes** | No | No |

---

## 2. Code Complexity Comparison

| Metric | Option A | Option B | Option C |
|---|---|---|---|
| Go source lines | 166 | 137 | 155 |
| Resource definitions | 1 | 1 | 2 |
| CRUD function sets | 1 (4 functions) | 1 (4 functions) | 2 (8 functions) |
| Custom validation code | `validateScheduleCombination()` — 45 lines | `optionBCustomizeDiff()` — 15 lines | None (schema handles it) |
| Schema complexity | Simple (all Optional) | Moderate (ConflictsWith, ValidateFunc, CustomizeDiff) | Simple (Required fields, ValidateFunc) |
| Provider registration | 1 resource | 1 resource | 2 resources |

### Observations

- **Option A** has the most custom validation code (the `validateScheduleCombination` function is 45 lines). All validation logic lives in Go and must be duplicated between Create and Update.
- **Option B** is the most concise overall (137 lines) and leverages SDK primitives. The `CustomizeDiff` function is only 15 lines.
- **Option C** has more CRUD boilerplate (8 functions vs 4) but **zero** custom validation logic for the conditional requirement. The `sharedFields()` helper keeps schema DRY.

---

## 3. UX Comparison — User's `.tf` File

### Option A & B (single resource)

```hcl
# Recurring
resource "devin_schedule" "daily" {
  title         = "Daily standup"
  prompt        = "Summarise open PRs"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
}

# One-time
resource "devin_schedule" "deploy" {
  title         = "Launch deploy"
  prompt        = "Deploy v2.1"
  schedule_type = "one_time"
  scheduled_at  = "2025-06-01T09:00:00Z"
}
```

- User must remember which attribute pairs with which `schedule_type`.
- Autocomplete shows both `frequency` and `scheduled_at` regardless of `schedule_type`.
- `for_each` loops can create both types from the same resource block (flexible but error-prone).

### Option C (two resources)

```hcl
# Recurring
resource "devin_recurring_schedule" "daily" {
  title     = "Daily standup"
  prompt    = "Summarise open PRs"
  frequency = "0 9 * * 1-5"
}

# One-time
resource "devin_one_time_schedule" "deploy" {
  title        = "Launch deploy"
  prompt       = "Deploy v2.1"
  scheduled_at = "2025-06-01T09:00:00Z"
}
```

- No `schedule_type` field needed — the resource name makes intent explicit.
- Autocomplete only shows relevant attributes (no `scheduled_at` on `devin_recurring_schedule`).
- `terraform state list` instantly shows which schedules are recurring vs one-time.
- `for_each` requires separate blocks for each type (cleaner but less flexible for mixed lists).

### UX Verdict

Option C is the clearest for users: the resource name communicates intent, irrelevant attributes don't appear, and the Terraform state is self-documenting. Options A/B are more concise when users manage many schedules of mixed types through a single `for_each`.

---

## 4. How Other Providers Handle Similar Patterns

### `aws_cloudwatch_event_rule`

The AWS provider uses the **Option B** pattern. `schedule_expression` (cron/rate) and `event_pattern` (JSON) are both Optional, but a `CustomizeDiff` enforces that at least one is set:

```go
CustomizeDiff: customdiff.Sequence(
    func(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
        if diff.Get("schedule_expression").(string) == "" &&
           diff.Get("event_pattern").(string) == "" {
            return errors.New("one of 'event_pattern' or 'schedule_expression' must be provided")
        }
        return nil
    },
),
```

### `aws_db_instance` (engine vs snapshot)

Uses **ConflictsWith** between `engine`/`snapshot_identifier` — you provide one or the other. This is the same SDK-level approach as Option B.

### `google_compute_instance` (boot disk)

Uses the **Option C split-resource** pattern: `google_compute_instance` vs `google_compute_instance_from_machine_image`. Each has exactly the fields it needs.

### `aws_s3_bucket` → split in v4

AWS provider v4 famously **split** `aws_s3_bucket` into multiple resources (`aws_s3_bucket_versioning`, `aws_s3_bucket_acl`, etc.) to reduce conditional complexity. This mirrors the Option C philosophy.

### Pattern Summary

| Provider / Resource | Pattern | Closest Option |
|---|---|---|
| `aws_cloudwatch_event_rule` | ConflictsWith + CustomizeDiff | **Option B** |
| `aws_db_instance` | ConflictsWith | **Option B** |
| `google_compute_instance` / `…_from_machine_image` | Split resources | **Option C** |
| `aws_s3_bucket` v3 → v4 | Migrated from single to split | **Option C** |

The industry trend is toward Option B for simple mutual exclusion and Option C when the sub-types have meaningfully different schemas.

---

## 5. Test Results

All 18 acceptance tests pass across the three options:

```
=== RUN   TestOptionA_RecurringValid            --- PASS
=== RUN   TestOptionA_OneTimeValid              --- PASS
=== RUN   TestOptionA_RecurringMissingFrequency --- PASS
=== RUN   TestOptionA_OneTimeMissingScheduledAt --- PASS
=== RUN   TestOptionA_RecurringWithScheduledAt  --- PASS
=== RUN   TestOptionB_RecurringValid            --- PASS
=== RUN   TestOptionB_OneTimeValid              --- PASS
=== RUN   TestOptionB_ConflictsBothSet          --- PASS
=== RUN   TestOptionB_RecurringMissingFrequency --- PASS
=== RUN   TestOptionB_OneTimeMissingScheduledAt --- PASS
=== RUN   TestOptionB_InvalidCron               --- PASS
=== RUN   TestOptionB_InvalidDatetime           --- PASS
=== RUN   TestOptionC_RecurringValid            --- PASS
=== RUN   TestOptionC_OneTimeValid              --- PASS
=== RUN   TestOptionC_RecurringMissingFrequency --- PASS
=== RUN   TestOptionC_OneTimeMissingScheduledAt --- PASS
=== RUN   TestOptionC_InvalidCron               --- PASS
=== RUN   TestOptionC_InvalidDatetime           --- PASS
PASS  (5.8s)
```

---

## 6. Recommendation

**Recommended: Option B (SDK validators)** with a note on when to reconsider.

### Why Option B

1. **Plan-time errors.** Users get immediate feedback without waiting for apply or burning API calls. This is the single biggest differentiator vs Option A.

2. **Single resource, simpler API surface.** Users learn one resource name (`devin_schedule`). Documentation, examples, and `for_each` patterns are simpler with one resource.

3. **Industry precedent.** The AWS provider — the most widely used Terraform provider — uses this exact pattern for `aws_cloudwatch_event_rule` and `aws_db_instance`. Users are already familiar with it.

4. **Least code.** At 137 lines, it is the most concise implementation. The `CustomizeDiff` is 15 lines. ConflictsWith and ValidateFunc are declarative and require no custom logic.

5. **Future-proof.** If a third `schedule_type` is added later (e.g., `event_driven`), extending the `CustomizeDiff` is a 5-line change. With Option C, a new resource type would be needed.

### When to reconsider (Option C)

If the recurring and one-time schedule types diverge significantly in the future (e.g., recurring schedules gain `timezone`, `max_concurrent_runs`, `pause_windows` while one-time gains `retry_after`, `expiry`), the single-resource schema will become cluttered with type-specific Optional fields. At that point, splitting into `devin_recurring_schedule` / `devin_one_time_schedule` (Option C) would be cleaner — the same evolution path AWS followed with `aws_s3_bucket`.

### Why not Option A

Option A should be avoided. Apply-time validation is strictly inferior: it wastes the user's time, makes `terraform plan` misleading, and goes against Terraform's design philosophy of catching errors as early as possible. The only scenario where CRUD-time validation is unavoidable is when validation depends on remote state (e.g., checking whether a referenced resource exists on the server). That does not apply here.

---

## File Index

| File | Description |
|---|---|
| `internal/provider/schedule_poc/option_a.go` | Option A: CRUD-time validation implementation |
| `internal/provider/schedule_poc/option_b.go` | Option B: SDK validators implementation |
| `internal/provider/schedule_poc/option_c.go` | Option C: Split resources implementation |
| `internal/provider/schedule_poc/option_a_test.go` | Option A acceptance tests (5 tests) |
| `internal/provider/schedule_poc/option_b_test.go` | Option B acceptance tests (7 tests) |
| `internal/provider/schedule_poc/option_c_test.go` | Option C acceptance tests (6 tests) |
| `internal/provider/schedule_poc/helpers_test.go` | Shared test helper |
| `examples/schedules_poc/option_a/main.tf` | Option A example Terraform config |
| `examples/schedules_poc/option_b/main.tf` | Option B example Terraform config |
| `examples/schedules_poc/option_c/main.tf` | Option C example Terraform config |
