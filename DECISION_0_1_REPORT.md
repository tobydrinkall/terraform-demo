# Decision 0.1: terraform-plugin-sdk/v2 vs terraform-plugin-framework

## Executive Summary

This report evaluates migrating the Devin Terraform provider from **terraform-plugin-sdk/v2** (the legacy SDK) to **terraform-plugin-framework** (the modern replacement). A working PoC of the `devin_knowledge_note` resource and `devin_knowledge_notes` data source was built using Framework and placed in `internal/framework_poc/`. A MUX example demonstrating hybrid serving is also included.

**Recommendation: Adopt terraform-plugin-framework for new development, using MUX to migrate incrementally.**

---

## 1. Architecture Comparison

```
┌─────────────────────────────────────────────────────────────────────┐
│                     SDK v2 Architecture                             │
│                                                                     │
│   main.go                                                           │
│     │                                                               │
│     ▼                                                               │
│   plugin.Serve()  ──── Protocol v5 (gRPC) ────  Terraform CLI       │
│     │                                                               │
│     ▼                                                               │
│   *schema.Provider                                                  │
│     ├── Schema: map[string]*schema.Schema     (flat key-value)      │
│     ├── ResourcesMap: map[string]*schema.Resource                   │
│     │     └── CRUD: func(ctx, *ResourceData, interface{}) diag      │
│     └── DataSourcesMap: map[string]*schema.Resource                 │
│                                                                     │
│   State access: d.Get("key").(type)  ← runtime type assertions     │
│   Validation:   ValidateFunc / ValidateDiagFunc                     │
│   Plan mods:    DiffSuppressFunc (limited)                          │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                   Framework Architecture                            │
│                                                                     │
│   main.go                                                           │
│     │                                                               │
│     ▼                                                               │
│   providerserver.Serve()  ── Protocol v6 (gRPC) ── Terraform CLI   │
│     │                                                               │
│     ▼                                                               │
│   provider.Provider (interface)                                     │
│     ├── Schema()      → schema.Schema          (strongly typed)     │
│     ├── Resources()   → []func() resource.Resource                  │
│     │     └── CRUD via interfaces:                                  │
│     │         resource.Resource                                     │
│     │         resource.ResourceWithConfigure                        │
│     │         resource.ResourceWithImportState                      │
│     │         resource.ResourceWithModifyPlan                       │
│     │         resource.ResourceWithValidateConfig                   │
│     └── DataSources() → []func() datasource.DataSource             │
│                                                                     │
│   State access: req.Plan.Get(ctx, &typedStruct)  ← compile-time    │
│   Validation:   schema/validator interfaces                         │
│   Plan mods:    planmodifier interfaces (composable)                │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Difference: Type Safety Pipeline

```
SDK v2 — Runtime type assertions (fails at runtime):
┌────────────┐    d.Get("name")    ┌──────────┐   .(string)    ┌──────────┐
│ HCL Config │ ──────────────────► │ interface│ ────────────► │  string  │
│            │                     │    {}    │  PANIC if     │          │
└────────────┘                     └──────────┘  wrong type   └──────────┘

Framework — Compile-time type safety (fails at compile time):
┌────────────┐  req.Plan.Get(ctx,  ┌──────────────────────────────────┐
│ HCL Config │  &model)            │  KnowledgeNoteResourceModel {    │
│            │ ──────────────────► │    Name    types.String          │
└────────────┘  Unmarshal via      │    Content types.String          │
                tfsdk struct tags  │    Size    types.Int64           │
                                   │  }                               │
                                   └──────────────────────────────────┘
                                   Compiler enforces field types ✓
```

---

## 2. Side-by-Side Line Counts

| File                       | SDK v2 (lines) | Framework (lines) | Delta  |
|----------------------------|:--------------:|:-----------------:|:------:|
| provider.go                |       51       |        91         | +78%   |
| resource_knowledge_note.go |       97       |       180         | +86%   |
| data_source_knowledge_notes.go | 96         |       153         | +59%   |
| **Total (core)**           |    **244**     |     **424**       | **+74%** |
| mux_example.go (bonus)     |       —        |        70         |   —    |

> The Framework code is ~74% more lines. However, this is **front-loaded verbosity** — the extra lines are type definitions (structs), interface implementations, and explicit plan modifiers that replace implicit SDK magic. As the provider grows, the per-resource overhead amortizes.

---

## 3. Feature Comparison

### 3.1 Schema Definition

**SDK v2** — Flat map with string keys, schema introspected at runtime:
```go
// SDK: 14 lines for the "scope" field
"scope": {
    Type:         schema.TypeString,
    Optional:     true,
    Default:      "organization",
    ValidateFunc: validation.StringInSlice(
        []string{"organization", "user", "repo"}, false,
    ),
    Description:  "The visibility scope...",
},
```

**Framework** — Strongly typed attribute objects with dedicated types:
```go
// Framework: 14 lines for the "scope" field
"scope": schema.StringAttribute{
    Optional:    true,
    Computed:    true,
    Default:     stringdefault.StaticString("organization"),
    Description: "The visibility scope...",
    Validators: []validator.String{
        stringvalidator.OneOf("organization", "user", "repo"),
    },
},
```

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| Per-field verbosity | Similar | Similar |
| Type errors | Runtime panic | Compile error |
| IDE autocomplete | Limited (map values) | Full (struct fields) |

### 3.2 Validators

```
SDK v2 Validators:
┌───────────────────────────────────────┐
│ ValidateFunc: func(val interface{},   │
│               key string) ([]string,  │
│               []error)                │  ← Untyped input
│                                       │  ← Returns (warns, errs)
│ ValidateDiagFunc: func(val interface│  ← Slightly better
│                   {}, path cty.Path)  │  ← Still untyped
│                   diag.Diagnostics    │
└───────────────────────────────────────┘
  Helpers: validation.StringInSlice(),
           validation.StringLenBetween()

Framework Validators:
┌───────────────────────────────────────┐
│ Validators: []validator.String{       │
│   stringvalidator.OneOf("a", "b"),    │  ← Type-safe
│   stringvalidator.LengthBetween(1,n), │  ← Composable
│   stringvalidator.RegexMatches(...),  │  ← Chainable
│   stringvalidator.ConflictsWith(...), │  ← Cross-field
│ }                                     │
└───────────────────────────────────────┘
  Also: validator.Int64, validator.Bool,
        validator.List, validator.Object
  Custom: implement validator.String interface
```

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| Type safety of validator input | `interface{}` | Typed (e.g., `string`) |
| Built-in validators | ~15 helpers | ~40+ per type |
| Cross-attribute validation | Manual in CustomizeDiff | `ConflictsWith`, `AlsoRequires`, `AtLeastOneOf` |
| Custom validators | `ValidateFunc` | Implement `validator.String` interface |
| Composability | Single function | Slice of validators (AND logic) |

### 3.3 Plan Modifiers

```
SDK v2 Plan Modification:
┌─────────────────────────────────────────────────────────┐
│ DiffSuppressFunc:    Suppress diffs (limited)           │
│ ForceNew:            Recreate on change (boolean flag)  │
│ CustomizeDiff:       Provider-level hook (global scope) │
│                                                         │
│ Problem: CustomizeDiff is one big function for the      │
│ entire resource, not per-attribute.                     │
└─────────────────────────────────────────────────────────┘

Framework Plan Modifiers:
┌─────────────────────────────────────────────────────────┐
│ PlanModifiers: []planmodifier.String{                   │
│   stringplanmodifier.UseStateForUnknown(),  ← per-attr  │
│   stringplanmodifier.RequiresReplace(),     ← ForceNew  │
│   stringplanmodifier.RequiresReplaceIf(f),  ← conditional│
│   MyCustomPlanModifier{},                  ← composable │
│ }                                                       │
│                                                         │
│ Each attribute gets its own list of plan modifiers.     │
│ Custom modifiers implement a typed interface.           │
└─────────────────────────────────────────────────────────┘
```

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| Granularity | Resource-level (`CustomizeDiff`) | Per-attribute |
| Built-in modifiers | `ForceNew`, `DiffSuppressFunc` | `UseStateForUnknown`, `RequiresReplace`, `RequiresReplaceIf`, `RequiresReplaceIfConfigured` |
| Custom modifiers | Callback function | Typed interface |
| Composability | Not composable | Slice of modifiers per attribute |

### 3.4 Sensitive Attributes

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| Mark as sensitive | `Sensitive: true` | `Sensitive: true` |
| WriteOnly attributes | Not supported | `WriteOnly: true` (v1.12+) — value never stored in state |
| Ephemeral resources | Not supported | Supported (v1.13+) |

```
WriteOnly (Framework-only feature):
┌────────────────────────────────────────────────────┐
│ "api_key": schema.StringAttribute{                 │
│     Required:  true,                               │
│     Sensitive: true,                               │
│     WriteOnly: true,   ← NEW in Framework          │
│ }                                                  │
│                                                    │
│ Value is sent to provider during Create/Update     │
│ but is NEVER persisted in Terraform state file.    │
│ Solves the "secrets in state" problem entirely.    │
└────────────────────────────────────────────────────┘
```

This is a **significant security advantage** — SDK v2 has no equivalent. Sensitive values in SDK are masked in CLI output but still stored in the state file.

### 3.5 Nested Objects / Complex Types

```
SDK v2 Nested Types:
┌────────────────────────────────────────────────────┐
│ "notes": {                                         │
│   Type: schema.TypeList,                           │
│   Elem: &schema.Resource{                          │
│     Schema: map[string]*schema.Schema{...}         │
│   },                                               │
│ }                                                  │
│                                                    │
│ Access: d.Get("notes").([]interface{})             │
│         item.(map[string]interface{})["name"]      │
│         ← Deeply nested type assertions            │
│         ← No compile-time safety                   │
└────────────────────────────────────────────────────┘

Framework Nested Types:
┌────────────────────────────────────────────────────┐
│ "notes": schema.ListNestedAttribute{               │
│   NestedObject: schema.NestedAttributeObject{      │
│     Attributes: map[string]schema.Attribute{...}   │
│   },                                               │
│ }                                                  │
│                                                    │
│ Access: model.Notes  (typed []NoteItemModel)       │
│         note.Name    (types.String)                │
│         ← Compile-time type safety                 │
│         ← IDE navigation works                     │
└────────────────────────────────────────────────────┘
```

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| Nested definition | `Elem: &schema.Resource{}` | `ListNestedAttribute` / `SingleNestedAttribute` |
| Access pattern | `interface{}` cascading assertions | Typed struct unmarshaling |
| Set semantics | `schema.TypeSet` (hash-based) | `types.Set` with explicit element type |
| Object type | Not natively supported | `schema.ObjectAttribute` with `AttributeTypes` |

### 3.6 State Management

```
SDK v2:                              Framework:
d.Set("name", "value")               resp.State.Set(ctx, &model)
d.Get("name").(string)               req.Plan.Get(ctx, &model)
d.SetId("abc")                       model.ID = types.StringValue("abc")
d.Id()                                model.ID.ValueString()

  │                                     │
  ▼                                     ▼
  Individual field get/set           Whole-struct serialization
  with type assertions               with compile-time types
```

---

## 4. MUX Feasibility

### Architecture

```
┌──────────────────────────────────────────────────────┐
│                  Terraform CLI                       │
│              (speaks Protocol v6)                    │
└──────────────────┬───────────────────────────────────┘
                   │ gRPC
┌──────────────────▼───────────────────────────────────┐
│              tf6muxserver                            │
│         (routes by resource type name)               │
│                                                      │
│   "devin_session"  ──► SDK v2 (via tf5to6server)     │
│   "devin_knowledge_note" ──► Framework (native v6)   │
│   "devin_playbook" ──► Framework (native v6)         │
└──────────────────────────────────────────────────────┘
```

### Implementation (see `internal/framework_poc/mux_example.go`)

The MUX approach requires:
1. **tf5to6server** — upgrades the SDK v2 provider (protocol v5) to protocol v6
2. **tf6muxserver** — combines multiple v6 providers into one server
3. Each resource type name must be unique across all providers in the MUX

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Setup complexity | **Low** — ~30 lines of MUX wiring in `main.go` |
| Runtime overhead | **Negligible** — routing is a simple map lookup |
| Constraints | Resource names must be unique across SDK and Framework providers |
| Provider schema | Both providers must have compatible provider-level schemas |
| Testing | Both SDK and Framework tests work independently |
| Migration path | Move resources one-at-a-time from SDK → Framework |

### Constraint: Provider Schema Overlap

The MUX server merges the provider-level schemas. Both the SDK provider and Framework provider define `api_key` and `base_url`. The MUX server validates that overlapping attributes have compatible definitions. In practice this works if both sides use the same types and descriptions, but it's a maintenance burden to keep them in sync.

**Mitigation**: During migration, strip resources out of the SDK provider as they move to Framework, eventually removing the SDK provider entirely.

---

## 5. Protocol Version Comparison

```
┌───────────────────────────────────────────────────────┐
│           Protocol v5 (SDK v2)                        │
│                                                       │
│  • Terraform 0.12+                                    │
│  • No WriteOnly attributes                            │
│  • No ephemeral resources                             │
│  • No deferred actions                                │
│  • Limited plan modification                          │
│  • No resource identity (move detection)              │
└───────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────┐
│           Protocol v6 (Framework)                     │
│                                                       │
│  • Terraform 1.0+ (required)                          │
│  • WriteOnly attributes (Terraform 1.11+)             │
│  • Ephemeral resources (Terraform 1.10+)              │
│  • Deferred actions (Terraform 1.9+)                  │
│  • Per-attribute plan modifiers                       │
│  • Resource identity / move detection                 │
│  • Better diagnostics with attribute paths            │
└───────────────────────────────────────────────────────┘
```

---

## 6. Migration Effort Matrix

| Phase | Effort | Description |
|-------|--------|-------------|
| 1. Add MUX + Framework dependency | 1 day | Wire up `tf6muxserver` in `main.go`, add deps |
| 2. Migrate `devin_knowledge_note` | 1–2 days | Port schema, CRUD, validators, tests |
| 3. Migrate `devin_knowledge_notes` data source | 0.5 day | Simpler — read-only |
| 4. Add new resources directly in Framework | Ongoing | Each new resource uses Framework from day 1 |
| 5. Remove SDK dependency | When all resources migrated | Delete SDK provider, simplify MUX |

---

## 7. Recommendation

### Adopt terraform-plugin-framework using incremental MUX migration

**Rationale:**

1. **terraform-plugin-sdk/v2 is in maintenance mode.** HashiCorp has stated no new features will be added. All new capabilities (WriteOnly, ephemeral resources, deferred actions) are Framework-only.

2. **Type safety prevents entire classes of bugs.** The SDK's `interface{}` type assertions are a common source of panics in production providers. Framework's typed models catch these at compile time.

3. **WriteOnly attributes solve the secrets-in-state problem.** For a Devin API provider that handles API keys, this is a material security improvement with no SDK equivalent.

4. **MUX makes migration zero-risk.** Existing SDK resources continue to work unchanged while new resources are built with Framework. No big-bang rewrite required.

5. **The verbosity cost is acceptable.** Framework code is ~74% more lines, but the extra lines are type definitions and explicit interfaces — they improve readability and maintainability. The per-resource overhead shrinks proportionally as CRUD logic grows.

6. **Community direction.** The majority of actively maintained providers (AWS, Azure, GCP) are migrating to Framework. Framework is the standard for all new HashiCorp-endorsed providers.

### Suggested Next Step

Implement MUX in `main.go`, keep the existing SDK `devin_knowledge_note` resource as-is, and build the next resource (`devin_session`) directly in Framework. Migrate `devin_knowledge_note` to Framework when convenient.

---

## Appendix: File Inventory

| Path | Description |
|------|-------------|
| `internal/provider/provider.go` | Existing SDK v2 provider (51 lines) |
| `internal/provider/resource_knowledge_note.go` | Existing SDK v2 resource (97 lines) |
| `internal/provider/data_source_knowledge_notes.go` | Existing SDK v2 data source (96 lines) |
| `internal/framework_poc/provider.go` | Framework provider PoC (91 lines) |
| `internal/framework_poc/resource_knowledge_note.go` | Framework resource PoC (180 lines) |
| `internal/framework_poc/data_source_knowledge_notes.go` | Framework data source PoC (153 lines) |
| `internal/framework_poc/mux_example.go` | MUX hybrid serving example (70 lines) |
