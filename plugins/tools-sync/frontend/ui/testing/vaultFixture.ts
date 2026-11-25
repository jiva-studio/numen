/** The transfer counts of a vault that moves nothing, which the core always sends. */
export const NO_TRANSFER = {
  syncing: false,
  syncedBytes: 0,
  totalBytes: 0,
  pendingFiles: 0,
  onDemandFiles: 0,
  onDemandBytes: 0,
} as const
