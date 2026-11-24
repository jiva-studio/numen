/** The words of the devices and of the link of a device. */
export const WORDS = {
  devices: 'Devices',
  deviceLinkNew: 'Link a new device',
  deviceJoin: 'Join account',
  deviceThis: 'This device',
  deviceRevoked: 'Revoked',
  deviceLinked: (date: string) => (date === '' ? 'Linked' : `Linked ${date}`),
  deviceRevoke: 'Remove',
  deviceRevokeTitle: 'Remove this device?',
  deviceRevokeWarning: (name: string) =>
    `${name} will lose access. Files on that device stay.`,
  deviceRevokeCancel: 'Cancel',
  linkNoCamera: 'No camera found. Paste the link from the other device.',
  linkCameraFailed: (reason: string) => `${reason} Paste the link from the other device.`,
  linkQr: 'QR code',
  linkPaste: 'Link from the other device',
  linkPasteInstead: 'Paste link instead',
  linkUse: 'Use link',
  linkCopy: 'Copy link',
  linkCopied: 'Copied',
  linkApprove: 'Link device',
  linkDecline: "Don't link",
  linkCodeField: 'Enter the code shown on your other device',
  linkCodeConfirm: 'Confirm',
  linkJoin: 'Join',
  linkCancel: 'Cancel',
  linkClose: 'Close',
  linkJoinTitle: 'Join account',
  linkHostTitle: 'Link a device',
  linkJoinHint: 'Point the camera at the code on your other device.',
  linkHostSteps: [
    'On the other device, tap Join account.',
    'Scan this code.',
    'Check the code matches, then tap Link.',
  ],
  linkCheckingCamera: 'Checking the camera',
  linkCheckingCameraDetail: 'Numen asks the system for the camera. Allow it if you are asked.',
  linkPreparingTitle: 'Preparing the link',
  linkPreparingDetail: 'Waiting for the Numen Sync server. This takes a few seconds.',
  linkStalledTitle: 'The link did not start',
  linkStalled:
    'Numen Sync cannot reach the server. Check the server address in the settings of Numen Sync.',
  linkRefusedTitle: 'The link did not start',
  linkWaitingScanTitle: 'Waiting for the other device to scan',
  linkWaitingTitle: 'Waiting for the other device',
  linkWaitingDetail: 'Keep this window open until the other device answers.',
  linkLinkingTitle: 'Linking the device',
  linkLinkingDetail: 'The keys of the account are being passed on.',
  linkWaitingCodeTitle: 'Waiting for the new device',
  linkWaitingCodeDetail: 'Type the confirmation code on the new device.',
  linkCodeHint: 'Type this confirmation code on the new device.',
  linkDoneTitle: 'The device is linked',
  linkDoneDetail: 'It has the account now.',
  linkFailedTitle: 'The link failed',
  linkChecking: 'Check that this is your new device.',
  linkTryAgain: 'Try again',
  linkDone: 'Done',
  linkDeviceName: 'Device name',
  linkDeviceFingerprint: 'Device fingerprint',
  linkCode: 'Confirmation code',
  linkCompare: 'Compare this fingerprint with the one the new device shows.',
  linkWaitingCode: 'Waiting for the code on the new device.',
  linkTriesLeft: (count: number) => `${count} tries left.`,
  linkJoinQuestion: (account: string) => `Join account ${account}?`,
  linkReason: (reason: string | undefined): string => {
    switch (reason) {
      case 'declined':
        return 'The new device was declined.'
      case 'deviceHasIdentity':
        return 'This device has an identity.'
      case 'tooManyWrongCodes':
        return 'Too many wrong codes. The link is closed.'
      case 'wrongCode':
        return 'The code was wrong.'
      case 'cancelled':
        return 'The link was cancelled.'
      case 'expired':
        return 'The link ran out of time.'
      case 'staleChain':
        return 'The account changed while linking. Try again.'
      default:
        return 'The link failed.'
    }
  },
} as const
