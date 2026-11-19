# Numen Sync

Synchronizes your vaults across devices when the **Numen Sync** license is active on this machine.

## Title Bar Status

Numen Sync renders its status in the `titleBar.status` slot at the top-right of the editor window:

- While synchronizing after activation or startup, it displays **Syncing…** with a progress indicator.
- Once synchronized, it displays **Synced**.
- When the `sync` license is inactive, the indicator is hidden.
