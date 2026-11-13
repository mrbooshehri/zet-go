# Terminal design direction

Use a compact header, contextual content, and a persistent shortcut footer. Wide terminals can show navigation, notes, and a preview; medium terminals should omit navigation, and narrow terminals should show one focused view. Preserve readable contrast in dark and light terminals, identify focus with text as well as color, and render untrusted Markdown without interpreting terminal escapes.

```text
 ZET                         / Search   ? Help
 Navigation       Notes                 Preview
 Dashboard        20261113102030        # A note
 All notes        20261113103512        Body…
 Search
 n New  e Edit  / Search  Tab Focus  q Quit
```

Input fields own ordinary character keys. Escape closes an input or modal before navigating back; Ctrl+C exits. Empty states should suggest an actionable next step. Errors should be visible and retain context; loading should remain responsive. This is a design direction for the unimplemented TUI, not a screenshot.
