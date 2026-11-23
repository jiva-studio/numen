export const WORDS = {
  syncingLabel: 'Syncing…',
  syncingDetail: 'Synchronizing vault…',
  syncedLabel: 'Synced',
  syncedDetail: 'Vault is synchronized.',
  subscriptionTitle: 'Numen Sync',
  subscriptionLoading: 'Loading the subscription…',
  noSubscription: 'No subscription',
  noSubscriptionText: 'Buy a subscription to sync your vaults between your devices.',
  active: 'Active',
  activeText: (periodEnd: string): string => `Active until ${periodEnd}.`,
  ended: 'Ended',
  graceText: (periodEnd: string, writesEnd: string): string =>
    `The subscription ended on ${periodEnd}. Changes are sent until ${writesEnd}, then the vaults in the cloud can only be read.`,
  readOnly: 'Read only',
  readOnlyText: (periodEnd: string, deleteAt: string): string =>
    `The subscription ended on ${periodEnd}. The vaults in the cloud can only be read and changes stay on this device. The cloud deletes them on ${deleteAt}.`,
  deleted: 'Deleted',
  deletedText: 'The cloud deleted your vaults. Buy a subscription to sync again.',
  noIdentityBadge: 'No identity',
  noIdentityText: 'Create an identity to buy a subscription.',
  syncServerUnreachable: 'Could not reach the sync server',
  buy: 'Buy',
  waitingForPayment: 'Waiting for the payment…',
  paymentNotSeen: 'The payment is not seen yet. Open this page again after you pay.',
  vaultsTitle: 'Vaults',
  connectNew: 'Connect as a new synced vault',
  connectExisting: 'Connect to an existing synced vault',
  chooseRemote: 'Choose a synced vault',
  noRemoteVaults: 'No synced vaults in your account',
  closeRemote: 'Cancel',
  notesDiffer: (count: number): string =>
    count === 1
      ? '1 note differs from the synced vault'
      : ` notes differ from the synced vault`,
  confirmConnect: 'Connect',
  cancelConnect: 'Cancel',
  loading: 'Loading vaults…',
  stateNotConnected: 'Not connected',
  stateConnected: 'Connected',
  stateCopy: 'a copy of a connected vault',
  noIdentity: 'No identity on this device',
  syncedJustNow: 'Synced less than a minute ago',
  syncedMinutesAgo: (count: number): string =>
    `Synced ${count} ${count === 1 ? 'minute' : 'minutes'} ago`,
  conflicts: (count: number): string =>
    `${count} ${count === 1 ? 'note' : 'notes'} changed on two computers`,
  deletionsPausedSend: (count: number, total: number): string =>
    `Sync is paused: ${count} of ${total} files are removed`,
  deletionsPausedApply: (count: number, total: number): string =>
    `Sync is paused: ${count} of ${total} files were removed on another computer`,
  removeFiles: 'Remove the files',
  keepFiles: 'Keep the files',
} as const
