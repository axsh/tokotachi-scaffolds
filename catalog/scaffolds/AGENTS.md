# WARNING: READ-ONLY TEMPLATE DIRECTORY

> [!CAUTION]
> **All files under this `templates/` directory are scaffold templates.**
>
> - These files **do not work as-is**. They are raw data that only become functional after being expanded and processed by the scaffold tool, which performs variable substitution and file generation.
> - AI agents **MUST NOT** directly modify, edit, or delete any files under this directory.
> - AI agents **MUST NOT** build, run, or test any code under this directory.
> - AI agents **MUST NOT** use any files under this directory as reference implementations or copy sources.

## Purpose of This Directory

`templates/` stores scaffold templates for project generation. Each template contains placeholder variables (e.g., `{{ProjectName}}`) that are replaced with actual values during scaffold generation. The files here are inert on their own and serve no purpose outside of the scaffold pipeline.

## Structure

```
templates/
  AGENTS.md                    -- This file (AI agent notice)
  axsh/                        -- Organization-scoped templates
    go-kotoshiro-mcp-feature/
    go-standard-feature/
    go-standard-project/
    ...
  root/                        -- Root-scoped templates
    project-default/
    ...
  ...
```

## Rules for AI Agents

| Prohibited Action | Reason |
|---|---|
| Editing or modifying files | Breaks the templates |
| Running or building code | Will fail due to placeholder variables |
| Using as reference code | Contains template-specific syntax that leads to misunderstanding |
| Deleting or moving files | Disrupts template management |

**Treat this entire directory as strictly READ-ONLY.**
