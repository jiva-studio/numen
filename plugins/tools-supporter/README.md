# Supporter

Shows a badge in the title bar when the Supporter license is active.

<!-- more -->

## What You Can Do

- **See your supporter badge**: When the `supporter` license is active on this machine, the title bar shows a compact Supporter badge in the status area.
- **View a thank-you note**: Point at the badge to read the thank-you tooltip.

## How It Works

The plugin has no backend. Its frontend reads `plugin.licenses.has('supporter')` and listens to `plugin.licenses.onChange('supporter', ...)` from `@numen/plugin-sdk`. While the license is active, it draws the badge in the `titleBar.status` slot; otherwise it renders nothing.
