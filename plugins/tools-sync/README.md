# Numen Sync

Synchronizes your vaults across devices when you have a subscription. The plugin shows the state of synchronization at the top-right of the editor window, so you can see at a glance whether your vault is up to date.

<!-- more -->

## What You Can Do

- **Buy a subscription**: The page **Identity and licenses** has the card **Numen Sync**. A click on a plan row opens the payment page in your browser, and the card shows the subscription once you pay. While it waits, the card keeps its rows under a layer with **Cancel**, which stops waiting. There is no license key.
- **See the state of your vault**: The status shows whether the vault is still synchronizing or already synchronized.
- **Sign in with your identity**: Numen Sync signs in to the sync server with the identity of this device. There is no name to type and no password. With no identity on this device it warns **No identity on this device**, and it signs in once the identity is created.

## The subscription

The sync server tells where the subscription of your identity stands, and the card shows it: no badge and a row per plan to choose, the plan as the badge with the rows Plan, Renews and Storage, **Ended** (changes are still sent until a date), **Read only** (the vaults in the cloud can only be read, changes stay on this device, and the cloud deletes them on a date) or **Deleted**. The plans to choose are shown in every phase but active, where **Change plan** and **Manage subscription** are shown instead. **Change plan** lists every plan, the current one checked; choosing another one asks to confirm the switch with the plan now, the new plan, the charge for the rest of the period and the next payment. After a click on a plan row the card asks the server every 3 seconds for up to 10 minutes whether you paid.

To cancel, in the app or without it, see [Cancel Numen Sync](https://docs.numen.md/cancel-sync/).

When the server refuses a change, or the grace period of an ended subscription runs, the plugin warns on its page: **Sync needs a subscription**, **Subscription ended: changes are sent until a date** or **Subscription ended: changes stay on this device**. The warning goes away when a pass goes well with an active subscription.

How to cancel, change the card or get an invoice, with the app or without it (by the email you paid with), and what happens to the vaults when the paid time ends, is in the guide [Numen Sync subscription](../../apps/docs/src/content/docs/sync.md).

## Title Bar Status

Numen Sync renders its status in the `titleBar.status` slot at the top-right of the editor window:

- While it asks the backend for the first time after startup, it displays **Syncing…** with a progress indicator.
- Once the backend answers, it displays **Synced**.
- While the device has no identity, the indicator is hidden.

## Settings

- **Sync server**: The address of the server this device signs in to (`https://sync.numen.md` by default). A change takes effect without a restart.

## How It Works

The plugin stays running while it is enabled and this device has an identity, or is restoring or joining one: the core starts it with the application and starts it again when its process ends (`backend.lifecycle` is `persistent`), and starts and stops it as the identity appears and goes (`backend.requires` is `["identity"]`). Without an identity a call from the window fails with the code `requirement_unmet`, which the cards read as no identity. It asks the core to sign a challenge of the server with the key of this device (permission `identity`), so the key never reaches the plugin. The server opens a session for the device; the plugin uses the expiry it returns to schedule the next sign-in 60 seconds before it ends and keeps no session token, so it has none to give to authenticated sync requests. When the server cannot be reached or refuses the sign-in, the plugin warns about it on its page and tries again every 30 seconds. When the server needs another protocol version it warns **Update Numen**. The protocol is `libs/shared/syncprotocol`.

## Notes sync

The plugin connects a vault to a synced vault of the identity and keeps its markdown notes equal on every computer of the identity. The settings view Vaults lists the vaults of the installation, connects one as a new synced vault or to an existing one, and warns about a copy of a connected folder.

A pass over a vault pulls the changes of the server, applies each by the rule of three hashes (local, base, head) and never overwrites a note that both computers changed, then pushes the notes that changed here. The server holds only sealed blobs and sealed metadata.

State lives in the store folder of the plugin in each vault: `link.json` (the synced vault and its sealed key), `files.json` (the base and the head revision of each note), `feed.json` (the position in the file change feed), `notes.json` (the notes as the last look found them) and `status.json` (when the last pass ended and how many notes wait on a conflict).

## How fast changes arrive

The plugin keeps one stream open to the server (`WatchService.WatchChanges`) that carries only hints: the id of a vault and the seq it reached. A hint above the cursor of a connected vault starts a pass over that vault at once, so a note written on one computer reaches the others in seconds. The stream never carries a note and never writes; the pass reads the changes from its own cursor and trusts nothing else. When the stream ends, drops, or is silent for three heartbeats, the plugin opens it again after 1.5 to 3 seconds (a random wait, so devices that lost the stream together do not return together; it doubles up to 2 minutes while the server does not answer), and the first hints name every vault that changed meanwhile. A stream that does not work raises no warning: the pass at the interval (`syncIntervalSeconds`, 30 seconds by default) still runs and still warns about a server that is away.

## Stopping and deleting the cloud copy

The card Vaults lists every synced vault of the identity from the catalog (`ListRemoteVaults`), with its size from `GetVaultUsage` (`sizeBytes`), also one no device of this installation syncs: such a row reads **Not syncing** and has the item **Delete cloud copy**. **Stop syncing** asks **Keep the cloud copy** or **Delete the cloud copy** (`StopSyncingVault` with `deleteCloudCopy`); a row that is not synced here asks the same question in **Delete the cloud copy of <name>?** (`DeleteCloudVault`). Deleting runs `usecase/catalog` `DeleteCloudVault`: it closes each open share of the vault (`SHARE_LOG_KIND_CLOSE`, in `usecase/share` `CloseShares`), appends a catalog entry whose body has `removed`, calls `DeleteVault`, then forgets the links of this device. A server failure leaves the vault linked and the person reads its text. The catalog reader skips a vault whose newest entry is removed, and the first step of a pass (`RenameRemoteVault`) forgets the link of a vault another device removed. A vault that is not linked here has no share closed by this device: the server drops them with the vault.

## Sharing a folder by a link

A folder is shared read-only with another identity by a link, in three steps: the sharer makes the link (`StartShareInvitation`), the recipient pastes it (`AcceptShareInvitation`), and both show the same six-digit code; the sharer then allows (`ApproveShareRequest`) or declines (`DeclineShareRequest`). Allowing runs the add-reader use case of `usecase/share` with the identity the device sent, then sends the recipient the invitation over the same channel, which the recipient's plugin accepts at once. An empty folder is the whole vault: it is shared like any folder, every file of the vault is rewrapped, and it is refused (`ErrHoldsShare`, naming the share) while a folder inside it is shared, as a folder is refused (`ErrInsideShare`) while the vault is shared.

The link is `numen-share:` and the base64url of `{v, p, f, i, e}` plus four bytes of its SHA-256: `p` is the text of a device-link payload (relay channel id, one-time X25519 public key, a 16-byte link secret, server origin), `f` the folder name (the vault name for the whole vault), `i` the fingerprint of the sharer identity, `e` the expiry in unix seconds. The handshake is the device-link handshake of `libs/shared/identity` (`LinkDisplayer` is the sharer, `LinkScanner` the recipient); the code is `LinkChannel.Code()`, derived from the keys of the channel, so a relay in the middle of the two sides makes the codes differ. The recipient sends its identity chain and device name sealed to the sharer after the hello, and nothing secret travels in the link besides the one-time public key and the link secret. A link works for one device (a second one gets a plain marker and reports "used"), until the relay ends the channel and 24 hours at most. The sharer keeps its links and requests in memory (`usecase/shareinvite` hub), so they end when the plugin stops. While a request waits the plugin reports the health item `share-request-waiting`.

## Errors the person reads

The person never reads a file name, an id or a Go error chain. `backend/wire/errorwire` is the one place where a failure of a method becomes a code and a text: every known failure (a missing vault folder, an unreachable server, a plan that does not allow syncing, a full storage, no identity) has a plain sentence in its table that says what happened and what to do, and any other error gets one generic sentence while its full chain goes to the log of the plugin as a warning with the name of the method. A new typed failure that a person can meet gets a row in that table.
