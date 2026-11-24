/** The words of the devices and of the link of a device. */
export const WORDS = {
  devices: 'Devices',
  deviceLinkNew: 'Link a new device',
  deviceJoin: 'Use existing',
  deviceThis: 'This device',
  deviceMenu: (name: string) => `Actions for ${name}`,
  deviceLinked: (date: string) => (date === '' ? 'Linked' : `Linked ${date}`),
  deviceRevoke: 'Remove',
  deviceRevokeTitle: 'Remove this device?',
  deviceRevokeWarning: (name: string, fingerprint: string) =>
    `${name} (${fingerprint}) will lose access. The files on the device stay.`,
  deviceRevokeCancel: 'Cancel',
  linkQr: 'QR code',
  linkPaste: 'Link from the other device',
  linkUse: 'Use link',
  linkCopy: 'Copy link',
  linkCopied: 'Copied',
  linkApprove: 'Link device',
  linkDecline: "Don't link",
  linkJoin: 'Use identity',
  linkCancel: 'Cancel',
  linkClose: 'Close',
  linkJoinTitle: 'Use existing',
  linkHostTitle: 'Link a device',
  linkJoinHint: 'Scan the code shown on your other device.',
  linkJoinPasteHint: 'Paste the link shown on your other device.',
  linkHostSteps: [
    'On this device, choose Link a device.',
    'On the other device, choose Use existing and scan this code.',
    'Approve the other device here.',
  ],
  linkCheckingCamera: 'Checking the camera',
  linkCheckingCameraDetail: 'Numen asks the system for the camera. Allow it if you are asked.',
  linkPreparingTitle: 'Preparing the link',
  linkPreparingDetail: 'Waiting for the Numen Sync server. This takes a few seconds.',
  linkStalledTitle: 'The link did not start',
  linkStalled:
    'Numen Sync cannot reach the server. Check the server address in the settings of Numen Sync.',
  linkRefusedTitle: 'The link did not start',
  linkWaitingTitle: 'Waiting for the other device',
  linkWaitingDetail: 'Keep this window open until the other device answers.',
  linkLinkingTitle: 'Linking the device',
  linkLinkingDetail: 'The keys of the identity are being passed on.',
  linkDoneTitle: 'The device is linked',
  linkDoneDetail: 'It has the identity now.',
  linkFailedTitle: 'The link failed',
  linkChecking: 'Make sure this is your new device.',
  linkTryAgain: 'Try again',
  linkDone: 'Done',
  linkDeviceName: 'Device name',
  linkDeviceFingerprint: 'Device fingerprint',
  linkCompare: 'Compare this fingerprint with the one the new device shows.',
  linkJoinQuestion: (account: string) => `Use identity ${account} on this device?`,
  linkReason: (reason: string | undefined): string => {
    switch (reason) {
      case 'declined':
        return 'The new device was declined.'
      case 'deviceHasIdentity':
        return 'This device has an identity.'
      case 'cancelled':
        return 'The link was cancelled.'
      case 'expired':
        return 'The link ran out of time.'
      case 'staleChain':
        return 'The identity changed while linking. Try again.'
      default:
        return 'The link failed.'
    }
  },
} as const
