/** The words of the devices and of the link of a device. */
export const WORDS = {
  devices: 'Devices',
  deviceLinkNew: 'Link a new device',
  deviceJoin: 'Join with a QR code',
  deviceThis: 'This device',
  deviceRevoked: 'Revoked',
  deviceLinked: (date: string) => (date === '' ? 'Linked' : `Linked ${date}`),
  deviceRevoke: 'Revoke',
  deviceRevokeTitle: 'Revoke this device?',
  deviceRevokeWarning: (name: string) =>
    `${name} will lose access. New notes will not be readable on it. Its files stay on it.`,
  deviceRevokeCancel: 'Cancel',
  account: 'Account fingerprint',
  linkNoCamera: 'No camera found: paste the code from the other device.',
  linkCameraFailed: (reason: string) => `${reason} Paste the code from the other device.`,
  linkQr: 'QR code',
  linkPaste: 'Paste the link text',
  linkUse: 'Use link',
  linkText: 'Link text',
  linkApprove: 'Link this device',
  linkDecline: 'Decline',
  linkCodeField: 'Enter the code shown on your other device',
  linkCodeConfirm: 'Confirm',
  linkJoin: 'Join',
  linkCancel: 'Cancel',
  linkClose: 'Close',
  linkJoinTitle: 'Join your account on this device',
  linkHostTitle: 'Link a new device to your account',
  linkJoinHint:
    'This device has no account yet. Link it to the account of your other device. The 12 recovery words are not needed.',
  linkHostHint: 'Add a new device to your account. The 12 recovery words are not needed.',
  linkSteps: (role: 'host' | 'joiner', mode: 'show' | 'scan'): readonly string[] => {
    const check =
      role === 'host'
        ? 'Check the name and fingerprint of the new device here, press Link this device, then type the confirmation code shown here on the new device.'
        : 'The other device shows a confirmation code after you link: type it on this device.'
    if (role === 'host') {
      return [
        'On the new device open Settings, Identity and choose Join with a QR code.',
        mode === 'show'
          ? 'Scan the QR code below with its camera, or paste the link text there.'
          : 'It shows a QR code: scan it with the camera of this device, or paste its link text below.',
        check,
      ]
    }
    return [
      'On your other device open Settings, Identity and choose Link a new device.',
      mode === 'show'
        ? 'Choose Scan a QR code there and point its camera at the QR code below, or paste the link text there.'
        : 'It shows a QR code: scan it with the camera of this device, or paste its link text below.',
      check,
    ]
  },
  linkScanInstead: 'Scan a QR code instead',
  linkShowInstead: 'Show a QR code instead',
  linkCheckingCamera: 'Checking the camera',
  linkCheckingCameraDetail: 'Numen asks the system for the camera. Allow it if you are asked.',
  linkPointCamera: 'Point the camera at the QR code on your other device',
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
