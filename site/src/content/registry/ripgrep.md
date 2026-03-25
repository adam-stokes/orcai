---
name: "ripgrep"
description: "Fast recursive search using rg"
tier: 2
command: "rg"
capabilities: ["search", "grep", "files"]
repo: "https://github.com/BurntSushi/ripgrep"
---

Wrap ripgrep as a Tier 2 orcai plugin via sidecar YAML.
Runs `rg --json` and pipes structured output into pipelines.

## Sidecar Config

```yaml
name: ripgrep
command: rg
args:
  - --json
  - "{{query}}"
```

## Usage

```yaml
steps:
  - id: search
    provider: ripgrep
    input:
      query: "TODO"
```
