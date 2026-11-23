# Numen Sync

Synchronizes your vaults across devices when the **Numen Sync** license is active on this machine. The plugin shows the state of synchronization at the top-right of the editor window, so you can see at a glance whether your vault is up to date.

<!-- more -->

## What You Can Do

- **Turn sync on with a license**: Activate the **Numen Sync** license for this machine and the plugin starts. It watches the license, so activating or removing it takes effect without a restart.
- **See the state of your vault**: The status shows whether the vault is still synchronizing or already synchronized.
- **Sign in with your identity**: Numen Sync signs in to the sync server with the identity of this device. There is no account name and no password. With no identity on this device it warns **No identity on this device**, and it signs in once the identity is created.

## Title Bar Status

Numen Sync renders its status in the `titleBar.status` slot at the top-right of the editor window:

- While synchronizing after activation or startup, it displays **Syncing…** with a progress indicator.
- Once synchronized, it displays **Synced**.
- When the `sync` license is inactive, the indicator is hidden.

## Settings

- **Sync server**: The address of the server this device signs in to (`https://sync.numen.md` by default). A change takes effect without a restart.

## How It Works

The plugin stays running while it is enabled: the core starts it with the application and starts it again when its process ends (`backend.lifecycle` is `persistent`). It asks the core to sign a challenge of the server with the key of this device (permission `identity`), so the key never reaches the plugin. The server opens a session for the device; the plugin uses the expiry it returns to schedule the next sign-in 60 seconds before it ends and keeps no session token, so it has none to give to authenticated sync requests. When the server cannot be reached or refuses the sign-in, the plugin warns about it on its page and tries again every 30 seconds. When the server needs another protocol version it warns **Update Numen**. The protocol is `libs/shared/syncprotocol`; `backend/cmd/fakesync` serves the stand-in server of `libs/shared/syncprotocol/fakeserver` for tests.
