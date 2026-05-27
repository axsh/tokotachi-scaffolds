# Phases

This directory organizes specifications and plans by project phase.

## Structure

```
phases/
  000-foundation/         # Phase 0: Foundation
    branches/
      <branch-name>/      # Git branch name
        ideas/            # Specification documents
        plans/            # Implementation plans
  001-<next-phase>/       # Phase 1: ...
    branches/
      <branch-name>/
        ideas/
        plans/
```

## Workflow

1. Create a specification in `branches/<branch-name>/ideas/`
2. Review and approve the specification
3. Create an implementation plan in `branches/<branch-name>/plans/`
4. Review and approve the plan
5. Execute the plan
