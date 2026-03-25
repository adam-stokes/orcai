## Why

The welcome widget currently exits on any keypress ("press any key to continue"), making it disposable noise rather than a useful home screen. New users have no idea what they can do or how to do it. With the new window/pane chord bindings added in `sensible-window-defaults`, the help text is also stale. The welcome screen should be a persistent, informative landing page that stays open until the user explicitly closes it.

## What Changes

- **Remove "any key to quit" behavior**: only `q`, `esc`, or `ctrl+c` close the welcome widget
- **Update help/instruction text**: add `c` (new shell window), `|`/`-` (splits), and arrow key navigation to the chord reference; reframe as onboarding rather than a transient splash
- **Remove "any key continue" footer line**: replace with `q / esc  close` hint
- **Update `enter` action copy**: currently implied by a footer line; make explicit in the body as "start a new session"
- Update the `buildHelp` footer string in `internal/welcome/welcome.go`

## Capabilities

### New Capabilities

*(none — this change only modifies existing behavior)*

### Modified Capabilities

- `welcome-dashboard`: Quit behavior changes from "any keypress exits" to "only q/esc/ctrl+c exits"; help text gains new window and pane chord entries; footer updated to show close hint instead of "any key continue"

## Impact

- `internal/welcome/welcome.go`: `Update` method and `buildHelp` function
- No API, bus, or config changes
- No breaking changes
