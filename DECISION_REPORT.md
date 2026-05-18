# Decision Report: Documentation Generation for Terraform Provider

## Executive Summary

I built all three documentation approaches for the Devin API v3 Terraform provider and evaluated them across seven criteria. **Recommendation: Option C (tfplugindocs with template overrides)** — it delivers the best balance of automation, quality, and maintainability.

---

## What Was Built

For each option, I produced:
- `docs/index.md` — provider overview
- `docs/resources/knowledge_note.md` — resource doc for `devin_knowledge_note`
- `docs/data-sources/knowledge_notes.md` — data source doc for `devin_knowledge_notes`

The provider has a realistic schema with rich `Description` fields on every attribute, examples in `examples/`, and proper Go build.

---

## Option A: tfplugindocs (Pure Auto-Generation)

### Setup
```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs generate --provider-name devin
```

### What It Produces
- Schema sections are auto-generated from Go `Description` fields
- Examples are pulled from `examples/resources/<name>/resource.tf`
- Frontmatter is auto-generated

### Strengths
- **Zero ongoing doc maintenance** — schema changes are reflected automatically
- **Guaranteed accuracy** — docs always match the code
- **Fast setup** — ~10 minutes to write examples and run the generator

### Weaknesses
- **No narrative context** — the provider index page has no authentication guidance, no "when to use this" section, no rate limiting notes
- **No import instructions** — users won't know how to `terraform import`
- **No API reference** — no mapping to underlying REST endpoints
- **Nested schema descriptions are stripped** — the data source's `notes` block loses per-attribute descriptions (see generated output: just bare type annotations)
- **No subcategory tagging** — the `subcategory` field is blank, so the Registry sidebar is unorganized

### Generated Output (Provider Index)
```markdown
# devin Provider

## Example Usage
[example code]

## Schema
### Required
- `api_key` (String, Sensitive) The API key for authenticating...
### Optional
- `base_url` (String) The base URL of the Devin API...
```

That's the *entire* provider page. No introduction, no authentication guidance, no links to resources.

---

## Option B: Hand-Written Markdown

### Setup
Manually create files in `docs/` following the [Terraform Registry format](https://developer.hashicorp.com/terraform/registry/providers/docs).

### What It Produces
- Rich narrative with authentication guidance, rate limiting, "when to use this" sections
- API reference tables mapping CRUD to REST endpoints
- Import instructions
- Advanced examples (e.g., `for_each` bulk provisioning)
- Behavioral notes (eventual consistency, semantic matching)

### Strengths
- **Maximum quality** — every section can be tailored to the user's needs
- **Rich context** — API reference, behavioral notes, advanced patterns
- **Full control** — subcategories, custom sections, conditional logic examples

### Weaknesses
- **Docs drift from implementation** — if you add a new attribute in Go but forget to update the markdown, docs are wrong. No CI check catches this without extra tooling.
- **High maintenance** — every new resource requires ~100-150 lines of hand-written markdown
- **Duplication** — schema descriptions exist in both Go code AND markdown, violating DRY
- **No automated preview** — no built-in `tfplugindocs validate` to catch structural issues

### Quality Comparison
The hand-written resource doc is ~120 lines vs. Option A's ~60 lines. The extra content includes:
- "When to Use This Resource" section
- `for_each` bulk provisioning example
- API Reference table with REST endpoints
- Import instructions
- Behavioral notes about eventual consistency

---

## Option C: tfplugindocs with Template Overrides (Hybrid)

### Setup
```bash
mkdir -p templates/resources templates/data-sources
# Write .md.tmpl files using Go template syntax
tfplugindocs generate --provider-name devin
```

### What It Produces
Templates define the *structure* (narrative sections, API reference, import instructions). The `{{ .SchemaMarkdown }}` directive injects auto-generated schema docs. Examples still come from `examples/` via `{{ tffile .ExampleFile }}`.

### How It Works
```
templates/resources/knowledge_note.md.tmpl
  ├── Hand-written narrative sections (When to Use, API Reference, Notes)
  ├── {{ .SchemaMarkdown }}  ← auto-generated from Go schema
  └── {{ tffile .ExampleFile }}  ← pulled from examples/
```

### Strengths
- **Schema accuracy guaranteed** — attribute lists always match the code
- **Rich narrative preserved** — custom sections (API Reference, Notes, Import) persist
- **Single source of truth** — descriptions live in Go code only, not duplicated in markdown
- **CI-friendly** — `tfplugindocs validate` catches drift between templates and schema
- **Scales well** — new resources get the same template structure; only the custom sections need writing

### Weaknesses
- **Template syntax learning curve** — Go templates (`.Name`, `.Type`, `.SchemaMarkdown`, `tffile`) require some familiarity
- **Slightly more setup** — ~15-20 minutes per resource (vs. ~5 min for Option A, ~30 min for Option B)
- **Nested schema descriptions still bare** — the `{{ .SchemaMarkdown }}` output for nested objects (like `notes` in the data source) strips per-attribute descriptions. This is a tfplugindocs limitation, not a template issue.

### Generated Output (Resource)
```markdown
# devin_knowledge_note (Resource)

[description from Go schema]

Knowledge notes are the primary mechanism for teaching Devin about your
organization's practices without repeating yourself in every prompt.

## When to Use This Resource
[hand-written narrative]

## Example Usage
[auto-pulled from examples/]

## Schema          ← auto-generated
### Required
- `content` (String) The body content...
### Optional
- `scope` (String) The visibility scope...
### Read-Only
- `id` (String) The unique identifier...

## Import          ← hand-written
## API Reference   ← hand-written
## Notes           ← hand-written
```

---

## Evaluation Matrix

| Criterion | Option A (Auto) | Option B (Hand-Written) | Option C (Hybrid) |
|---|---|---|---|
| **1. Initial Setup Effort** | ⬤ Low (~10 min) | ⬤⬤⬤ High (~2 hrs) | ⬤⬤ Medium (~45 min) |
| **2. Ongoing Maintenance** | ⬤ Near-zero | ⬤⬤⬤ High (manual per resource) | ⬤⬤ Low-medium (template per resource) |
| **3. Doc Quality** | ⬤ Bare minimum | ⬤⬤⬤ Excellent | ⬤⬤⬤ Excellent |
| **4. Consistency** | ⬤⬤⬤ Perfect (automated) | ⬤ Variable (human error) | ⬤⬤⬤ High (template-enforced) |
| **5. Developer Preview** | ⬤⬤⬤ `tfplugindocs generate` | ⬤ Manual (open in browser) | ⬤⬤⬤ `tfplugindocs generate` |
| **6. Drift Risk** | ⬤ None | ⬤⬤⬤ High | ⬤ None for schema, low for narrative |
| **7. Industry Adoption** | Used by small/new providers | Used by AWS (legacy `website/docs/`) | Used by GitHub, Grafana, MongoDB Atlas providers |

### Detailed Criterion Analysis

#### 1. Initial Setup Effort
- **A**: Write example `.tf` files, run generator. Done.
- **B**: Write 3 full markdown documents (~400 lines total) from scratch.
- **C**: Write 3 template files (~250 lines total), plus example `.tf` files.

#### 2. Ongoing Maintenance (Adding a New Resource)
- **A**: Write one example `.tf` file, run generator. ~5 minutes.
- **B**: Write ~120 lines of markdown, manually duplicating every schema attribute. ~30 minutes.
- **C**: Write one template (~80 lines), one example `.tf` file. Schema section auto-populates. ~15 minutes.

#### 3. Doc Quality
- **A**: Functional but sparse. No guidance beyond "here are the attributes." Users must read API docs separately.
- **B**: Best possible quality — tailored narrative, advanced examples, behavioral warnings.
- **C**: Nearly as good as B. The only gap is nested schema descriptions in data sources, which is a tfplugindocs limitation. All hand-written sections (API Reference, Notes, Import) are preserved.

#### 4. Consistency Across Resources
- **A**: Every resource doc looks identical. Perfectly consistent.
- **B**: Human variation. One resource might have Import instructions; another might not.
- **C**: Template enforces structure. Every resource gets the same sections. Only content varies.

#### 5. Developer Preview Workflow
- **A/C**: `tfplugindocs generate` → check `docs/` → commit. Can be added to CI.
- **B**: Open markdown files in a browser or viewer. No automated validation.

#### 6. Drift Risk
- **A**: Zero. Docs are regenerated from code.
- **B**: High. A developer adds `priority` attribute in Go but forgets to update `docs/resources/knowledge_note.md`. Docs are now wrong. No automated check catches this.
- **C**: Near-zero for schema (auto-generated). Low risk for narrative sections (they rarely change with schema updates). `tfplugindocs validate` catches structural issues in CI.

#### 7. What Top-Tier Providers Do
| Provider | Approach | Notes |
|---|---|---|
| **AWS** | Hand-written (`website/docs/`) | Legacy format, pre-dates tfplugindocs. 1000+ resources make migration impractical. |
| **Google Cloud** | Hand-written (generated from Magic Modules) | Custom code generation framework. |
| **Azure** | Hand-written + automation | Mixed approach due to scale. |
| **Datadog** | Auto-generated with script | Custom wrapper around tfplugindocs. |
| **GitHub** | tfplugindocs with validation | Uses `tfplugindocs validate` in CI. |
| **PagerDuty** | Hand-written in `website/docs/` | Legacy format. |

**Key insight**: Large legacy providers (AWS, GCP) use hand-written docs because they predate tfplugindocs. Modern and medium-sized providers overwhelmingly use tfplugindocs, often with template overrides for quality.

---

## Recommendation: Option C

**Use tfplugindocs with custom template overrides.**

### Why

1. **Schema accuracy is guaranteed** — the most critical category of doc drift (wrong attributes, missing types, stale defaults) is eliminated entirely.

2. **Quality approaches hand-written** — the narrative sections (authentication, API reference, import, behavioral notes) are just as rich as Option B, because you write them in templates.

3. **Maintenance scales** — when you add the 20th resource, you write one template and one example. The schema section auto-populates. With Option B, you'd be manually maintaining 20 argument reference tables.

4. **CI integration is trivial** — add `tfplugindocs validate` to your CI pipeline. If schema changes without regenerating docs, CI fails. Option B has no equivalent.

5. **Industry trajectory** — HashiCorp built tfplugindocs specifically because hand-written docs don't scale. Every new official provider uses it. Templates are the recommended extension mechanism.

### Key Trade-Off

The single most important trade-off is **automation vs. narrative quality**. Option A automates everything but produces sparse docs. Option B has perfect narrative but drifts. Option C threads the needle: auto-generated schema sections with hand-written narrative overlays, giving you both accuracy and quality.

### Concrete Next Steps

1. Keep the `templates/` directory with the three `.md.tmpl` files already created.
2. Add `tfplugindocs generate` to the Makefile:
   ```makefile
   docs:
   	tfplugindocs generate --provider-name devin
   
   docs-check: docs
   	@if [ "`git status --porcelain docs/`" ]; then \
   		echo "Docs out of date. Run 'make docs' and commit."; exit 1; \
   	fi
   ```
3. Add `make docs-check` to CI.
4. For each new resource: create `templates/resources/<name>.md.tmpl` with the standard sections, and `examples/resources/devin_<name>/resource.tf` with working examples.

---

## Appendix: File Inventory

### Files Created for This Evaluation

```
terraform-demo/
├── main.go                          # Provider entrypoint
├── internal/provider/
│   ├── provider.go                  # Provider schema (api_key, base_url)
│   ├── resource_knowledge_note.go   # Resource schema (9 attributes)
│   └── data_source_knowledge_notes.go # Data source schema (filters + nested notes)
├── examples/
│   ├── provider/provider.tf
│   ├── resources/devin_knowledge_note/resource.tf
│   └── data-sources/devin_knowledge_notes/data-source.tf
├── templates/                       # Option C templates
│   ├── index.md.tmpl
│   ├── resources/knowledge_note.md.tmpl
│   └── data-sources/knowledge_notes.md.tmpl
├── option_a_output/                 # Option A generated docs
│   ├── index.md
│   ├── resources/knowledge_note.md
│   └── data-sources/knowledge_notes.md
├── option_b_output/                 # Option B hand-written docs
│   ├── index.md
│   ├── resources/knowledge_note.md
│   └── data-sources/knowledge_notes.md
├── option_c_output/                 # Option C hybrid docs
│   ├── index.md
│   ├── resources/knowledge_note.md
│   └── data-sources/knowledge_notes.md
└── docs/                            # Current generated output (Option C)
```

### Line Counts

| File | Option A | Option B | Option C |
|---|---|---|---|
| `index.md` | 32 | 65 | 47 |
| `resources/knowledge_note.md` | 60 | 120 | 95 |
| `data-sources/knowledge_notes.md` | 66 | 100 | 86 |
| **Total** | **158** | **285** | **228** |

Option C produces 44% more content than Option A (all narrative value) while requiring zero duplicate maintenance of schema attributes.
