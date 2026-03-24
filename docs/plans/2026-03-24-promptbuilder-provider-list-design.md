# Prompt Builder Provider List — Design

## Goal

Replace the prompt builder's hardcoded `pluginList`/`modelsByPlugin` with the canonical picker provider list, giving it runtime-discovered providers (opencode, ollama + local models) identical to what the session picker shows.

## Section 1: picker.BuildProviders()

Export the existing `buildProviders()` as `picker.BuildProviders() []ProviderDef`, filtering out `shell` (not relevant for pipeline steps). Same runtime behaviour as the picker: filters by installed CLI, injects ollama models into ollama/opencode, creates ctx32k variants, writes opencode config.

## Section 2: BubbleModel data model

- `NewBubble(m *Model, providers []picker.ProviderDef) *BubbleModel`
- Remove package-level `pluginList []string` and `modelsByPlugin map[string][]string`
- Add `providers []picker.ProviderDef` field to `BubbleModel`
- Cycling uses `b.providers[b.pluginIndex].ID/.Label` and `b.providers[b.pluginIndex].Models`
- Skip `ModelOption.Separator == true` entries when cycling models
- `syncIndicesFromStep`, `applyPlugin`, `applyModel` updated to use `b.providers`
- `renderSelector` displays `provider.Label` (not raw ID)
- Tests replace `pluginList`/`modelsByPlugin` references with a local `testProviders` slice

## Section 3: run.go

- Call `picker.BuildProviders()` at startup
- Drive adapter registrations from the provider list (no hardcoded names)
- Pass providers to `NewBubble(m, providers)`
