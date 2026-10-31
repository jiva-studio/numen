# Update Notifier

Tells you when a newer build of Numen is published, and takes you to its download.

<!-- more -->

## What You Can Do

- **See that an update exists**: A version badge appears at the right end of the status bar while a newer build is published. Click it for the release date, the way to install on your system and a link to the release notes.
- **Dismiss an announcement**: **Dismiss** hides the badge for that version until you restart the window. A newer version shows it again.

The plugin checks quietly. A machine without a network connection, or a feed that does not answer, shows nothing and is asked again a little later.

## Settings

- **Channel**: **Stable** *(default)* announces releases; **Beta** announces builds published ahead of the next release.
- **Release feed address**: Address of the `latest.json` file to ask for the newest build. Empty asks the published feed of the channel, `https://dl.numen.md/latest/latest.json` for the stable channel.
- **Check interval**: Hours between two checks (`4` by default).

## How It Works

The plugin's backend requests the release feed and compares its build number with the running build; where the running build has no number, versions are compared. The window shows the result in the status bar. Its service `numen.updater` has the methods `Check` and `GetStatus`; a check runs only after the application has configured the plugin with its settings.
