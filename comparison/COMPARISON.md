# Terraform Provider Framework Comparison

## terraform-plugin-framework vs terraform-plugin-sdk v2

Comparison based on implementing the same `devin_knowledge_note` CRUD resource.

---

## Lines of Code

| Component                | Framework | SDK v2 |
|--------------------------|-----------|--------|
| Provider                 | 85        | 48     |
| Resource (CRUD + schema) | 275       | 213    |
| Tests                    | 30        | 73     |
| **Total (excl. shared)** | **390**   | **334**|

Framework is ~17% more code for the same functionality. The verbosity comes from
explicit typed models and interface implementations rather than schema-map + ResourceData.

---

## Architecture Comparison

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    terraform-plugin-framework                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐     ┌────────────────────────┐                        │
│  │ Provider     │     │ KnowledgeNoteResource  │                        │
│  │ (interface)  │     │ (interface)            │                        │
│  ├──────────────┤     ├────────────────────────┤                        │
│  │ Metadata()   │     │ Metadata()             │                        │
│  │ Schema()     │     │ Schema() ─► typed      │                        │
│  │ Configure()  │     │ Create(plan Model)     │                        │
│  │ Resources()  │     │ Read(state Model)      │                        │
│  │ DataSources()│     │ Update(plan+state)     │                        │
│  └──────────────┘     │ Delete(state Model)    │                        │
│                       │ ImportState()          │                        │
│                       └────────────────────────┘                        │
│                                                                         │
│  State Model (struct with tfsdk tags):                                  │
│  ┌───────────────────────────────────────────┐                          │
│  │ type KnowledgeNoteResourceModel struct {  │                          │
│  │   ID      types.String `tfsdk:"id"`       │  ◄── Compile-time safe   │
│  │   Name    types.String `tfsdk:"name"`     │                          │
│  │   ...                                     │                          │
│  │ }                                         │                          │
│  └───────────────────────────────────────────┘                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                    terraform-plugin-sdk v2                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────────┐     ┌──────────────────────────────┐          │
│  │ schema.Provider{     │     │ schema.Resource{             │          │
│  │   Schema: map[...]   │     │   Schema: map[string]*Schema │          │
│  │   ConfigureCtxFunc   │     │   CreateContext: func(...)   │          │
│  │   ResourcesMap       │     │   ReadContext: func(...)     │          │
│  │ }                    │     │   UpdateContext: func(...)   │          │
│  └──────────────────────┘     │   DeleteContext: func(...)   │          │
│                               │   Importer: {...}            │          │
│                               │   CustomizeDiff: func(...)   │          │
│  Data Access:                 │ }                            │          │
│  ┌──────────────────────────┐ └──────────────────────────────┘          │
│  │ d.Get("name").(string)  │  ◄── Runtime type assertion                │
│  │ d.Set("name", value)    │  ◄── String key, interface{} val           │
│  │ d.SetId(id)             │                                            │
│  └──────────────────────────┘                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Feature-by-Feature Comparison

### 1. Computed Attributes

| Aspect | Framework | SDK v2 |
|--------|-----------|--------|
| Declaration | `Computed: true` on schema attribute | `Computed: true` on schema field |
| Stability | `UseStateForUnknown()` plan modifier | Implicit — SDK preserves state |
| Access | `model.ID.ValueString()` | `d.Get("id").(string)` |

**Framework advantage**: `UseStateForUnknown()` explicitly documents intent — the value
won't change after creation. SDK achieves the same implicitly but less transparently.

### 2. Optional with Defaults

| Aspect | Framework | SDK v2 |
|--------|-----------|--------|
| Syntax | `Optional: true, Computed: true, Default: booldefault.StaticBool(true)` | `Optional: true, Default: true` |
| Null handling | `types.Bool` distinguishes null/unknown/value | `GetOk()` returns (value, wasSet) |
| Custom defaults | Implement `planmodifier` interface | Use `DefaultFunc` closure |

**SDK advantage**: Simpler one-liner for static defaults. Framework requires importing a
separate `booldefault` package and marking the attribute as both Optional + Computed.

### 3. Plan Modifiers vs CustomizeDiff

```go
// Framework: declarative plan modifiers per-attribute
"id": schema.StringAttribute{
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
}

// SDK v2: imperative CustomizeDiff function on the whole resource
CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
    // Access any field, make cross-field decisions
    return nil
}
```

| Aspect | Framework | SDK v2 |
|--------|-----------|--------|
| Scope | Per-attribute | Whole resource |
| Composability | Chain multiple modifiers per attr | Single function, manual dispatch |
| Cross-field logic | Requires custom plan modifier impl | Native — access any field |
| Testing | Unit-test individual modifiers | Test the whole diff function |

**Framework advantage**: Attribute-scoped modifiers are more maintainable and composable.
**SDK advantage**: Cross-field validation is trivial in a single function.

### 4. Import State

```go
// Framework
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// SDK v2
Importer: &schema.ResourceImporter{
    StateContext: schema.ImportStatePassthroughContext,
}
```

Both are one-liners for simple ID passthrough. Framework's approach allows targeting
specific attributes (not just `id`), which is useful for composite keys.

### 5. Type Safety

```
┌────────────────────────────────────────────────────────────────┐
│                     TYPE SAFETY SPECTRUM                        │
│                                                                │
│  SDK v2                                      Framework         │
│  ◄────────────────────────────────────────────────────►        │
│                                                                │
│  d.Get("nam").(string)    vs    model.Nam (compile error)      │
│       │                                │                       │
│       ▼                                ▼                       │
│  Runtime panic if key                 Won't compile if         │
│  typo or wrong type                   field doesn't exist      │
│                                                                │
│  d.Set("field", 42) on a             types.StringValue()       │
│  TypeString field ──► silent          won't accept int ──►     │
│  runtime error                        compile-time error       │
└────────────────────────────────────────────────────────────────┘
```

**Framework is significantly safer.** SDK v2's `interface{}` returns require type
assertions and string-keyed lookups — typos and type mismatches surface only at runtime.

### 6. Error Handling

| Aspect | Framework (`diag.Diagnostics`) | SDK v2 (`diag.Diagnostics`) |
|--------|-------------------------------|----------------------------|
| Package | `github.com/hashicorp/terraform-plugin-framework/diag` | `github.com/hashicorp/terraform-plugin-sdk/v2/diag` |
| Accumulation | `resp.Diagnostics.Append(...)` | Return `diag.Diagnostics` slice |
| Early exit | `if resp.Diagnostics.HasError() { return }` | `if diags.HasError() { return diags }` |
| Attribute path | Native `path.Root("field")` support | `diag.Diagnostic{AttributePath: ...}` |
| Warnings | `resp.Diagnostics.AddWarning(...)` | `diag.Diagnostic{Severity: diag.Warning}` |

Both use diagnostics, but Framework's approach is more ergonomic: it accumulates errors
on the response object rather than requiring explicit return-value chaining.

---

## Scaling to Complex/Nested Types (e.g., `devin_schedule`)

```
Hypothetical devin_schedule resource with nested blocks:
┌───────────────────────────────────────────────┐
│ schedule {                                    │
│   name = "daily-sync"                         │
│   cron = "0 9 * * *"                          │
│   session_config {          ◄── nested block  │
│     prompt    = "..."                         │
│     playbook  = "pb-123"                      │
│     repos     = ["org/repo"]  ◄── list type   │
│     env_vars  = { KEY = "val" } ◄── map type  │
│   }                                           │
│ }                                             │
└───────────────────────────────────────────────┘
```

| Aspect | Framework | SDK v2 |
|--------|-----------|--------|
| Nested block | `schema.SingleNestedAttribute{}` with typed inner model | `schema.Schema{ Elem: &schema.Resource{...} }` |
| List/Set attrs | `types.List` / `types.Set` with element type | `TypeList` / `TypeSet` with `Elem` |
| Access nested | `plan.SessionConfig.Prompt.ValueString()` | `d.Get("session_config.0.prompt").(string)` |
| Type model | Nested Go structs with `tfsdk` tags | Flattened access via dot-notation strings |
| Refactoring | Rename struct field → compiler finds all usages | Rename string key → grep + hope |

**Framework scales dramatically better.** Nested types in SDK v2 become deeply nested
`map[string]interface{}` with index-based access (`"block.0.field"`). Framework keeps
everything as strongly-typed nested structs.

---

## Long-Term Maintainability Assessment

| Factor | Framework | SDK v2 |
|--------|-----------|--------|
| HashiCorp investment | Active development, recommended for new providers | Maintenance mode only |
| Protocol support | Protocol 6 (latest) | Protocol 5 (legacy) |
| Mux compatibility | Native mux support | Requires `tf5to6server` bridge |
| Learning curve | Steeper (interfaces, plan modifiers, typed models) | Flatter (maps, functions) |
| Refactoring safety | High (compiler catches issues) | Low (runtime-only errors) |
| IDE support | Excellent (struct completion, type checking) | Poor (string keys, interface{}) |
| Documentation | Growing, but newer | Extensive, mature |
| Migration path | N/A (target) | → Framework (official guidance) |

---

## Recommendation

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ✦ Use terraform-plugin-framework for the Devin provider ✦      │
│                                                                  │
│  Reasons:                                                        │
│  1. SDK v2 is in maintenance mode — no new features             │
│  2. Type safety prevents entire classes of bugs                  │
│  3. Nested types (schedules, sessions) will be much cleaner      │
│  4. Plan modifiers compose better than monolithic CustomizeDiff  │
│  5. Protocol 6 enables future Terraform features                 │
│                                                                  │
│  The ~17% extra LOC is a worthwhile trade for:                   │
│  • Compile-time correctness                                      │
│  • Better IDE experience                                         │
│  • Future-proof architecture                                     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

The SDK v2 implementation is viable for quick prototypes, but for a production provider
managing multiple Devin resources (knowledge, playbooks, schedules, secrets), the
Framework's type safety and composability will save significant debugging time as the
provider grows.
