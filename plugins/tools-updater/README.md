# Update Notifier

Shows a notice when a newer build of Numen is published, with a link to its download.

<!-- more -->

## What You Can Do

- **See that an update exists**: A version badge appears at the right end of the status bar while a newer build is published. Click it for the release date, the way to install on your system and a link to the release notes.
- **Dismiss an announcement**: **Dismiss** hides the badge for that version until you restart the window. A newer version shows it again.

The plugin checks without showing messages. A machine without a network connection, or a feed that does not respond, shows nothing, and the feed is requested again later.

## Settings

- **Channel**: **Stable** _(default)_ notifies about releases; **Beta** notifies about builds published ahead of the next release.
- **Release feed address**: Address of the `latest.json` file to request the newest build. If empty, the published feed of the channel is used, `https://dl.numen.md/latest/latest.json` for the stable channel.
- **Check interval**: Hours between two checks (`4` by default).

## How It Works

The plugin's backend requests the release feed and compares its build number with the running build; where the running build has no number, versions are compared. The window shows the result in the status bar. Its service `md.numen.tools.updater` has the methods `Check` and `GetStatus`; a check runs only after the application has configured the plugin with its settings.
