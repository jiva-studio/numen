# Update Notifier

Shows a notice when a newer build of Numen is published, with a link to its download.

<!-- more -->

## What You Can Do

- **See that an update exists**: A version badge appears at the right end of the title bar while a newer build is published. Click it for the release date, the way to install on your system and a link to the release notes.
- **Dismiss an announcement**: **Dismiss** hides the badge for that version until you restart the window. A newer version shows it again.

The plugin checks without showing messages. A machine without a network connection, or a feed that does not respond, shows nothing, and the feed is requested again later.

## Settings

- **Channel**: **Stable** _(default)_ notifies about releases; **Beta** notifies about builds published ahead of the next release; **Custom** asks the release feed address below.
- **Release feed address**: Shown only for the **Custom** channel, which requires it: the address of the `latest.json` file to request the newest build. The Stable and Beta channels ask their own published feeds, and the stored address is kept while it is hidden.
- **Check interval**: Hours between two checks (`4` by default).

## How It Works

The plugin's backend requests the release feed and compares its build number with the running build; where the running build has no number, versions are compared. The window shows the result in the title bar. Its service `md.numen.tools.updater` has the methods `Check` and `GetStatus`; a check runs only after the application has configured the plugin with its settings.
