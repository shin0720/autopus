# Project Structure

```
autopus-adk-main/
    cmd/
        auto/
        generate-templates/
    configs/
    content/
        agents/
        hooks/
        methodology/
        profiles/
            executor/
        rules/
        skills/
    demo/
    docs/
    e2e/
        testdata/
    internal/
        cli/
            tui/
    pkg/
        adapter/
            claude/
            codex/
            gemini/
            opencode/
        arch/
        browse/
        config/
        connect/
        constraint/
        content/
        cost/
        detect/
        docs/
        e2e/
        experiment/
        hint/
        issue/
        learn/
        lore/
        lsp/
        orchestra/
        pipeline/
        search/
        selfupdate/
        setup/
        sigmap/
        spec/
        telemetry/
        template/
        terminal/
        version/
        worker/
            a2a/
            adapter/
            audit/
            auth/
            budget/
            compress/
            daemon/
            knowledge/
            mcp/
            mcpserver/
            net/
            parallel/
            pidlock/
            poll/
            qa/
            reaper/
            routing/
            scheduler/
            security/
            setup/
            stream/
            tui/
            workspace/
    scripts/
    templates/
        claude/
            commands/
            rules/
            skills/
        codex/
            agents/
            prompts/
            rules/
            skills/
        gemini/
            agents/
            commands/
            rules/
            settings/
            skills/
        hooks/
        shared/
```

## Directory Roles

- **cmd/** — CLI entry points
- **content/** — Content assets
- **docs/** — Documentation
- **internal/** — Private implementation packages
- **pkg/** — Public reusable libraries
  - **config/** — Configuration files
  - **content/** — Content assets
  - **docs/** — Documentation
- **scripts/** — Build and utility scripts
- **templates/** — Template files
