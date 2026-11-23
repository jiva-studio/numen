/** The words of the devices and of the link of a device. */
export const WORDS = {
  devices: 'Devices',
  deviceLinkNew: 'Link a new device',
  deviceJoin: 'Join with a QR code',
  deviceThis: 'This device',
  deviceRevoked: 'Revoked',
  deviceRevoke: 'Revoke',
  deviceRevokeTitle: 'Revoke this device?',
  deviceRevokeWarning: (name: string) =>
    `${name} will lose access. New notes will not be readable on it. Its files stay on it.`,
  deviceRevokeCancel: 'Cancel',
  account: 'Account fingerprint',
  linkShow: 'Show a QR code',
  linkScan: 'Scan a QR code',
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
  linkDeviceName: 'Device name',
  linkDeviceFingerprint: 'Device fingerprint',
  linkCode: 'Confirmation code',
  linkCompare: 'Compare this fingerprint with the one the new device shows.',
  linkWaiting: 'Waiting for the other device.',
  linkPreparing: 'Preparing the link.',
  linkLinking: 'Linking the device.',
  linkWaitingCode: 'Waiting for the code on the new device.',
  linkDone: 'The device is linked.',
  linkTriesLeft: (count: number) => `${count} tries left.`,
  linkJoinQuestion: (account: string) => `Join account ${account}?`,
  linkReason: (reason: string | undefined): string => {
    switch (reason) {
      case 'declined':
        return 'The new device was declined.'
      case 'device_has_identity':
        return 'This device has an identity.'
      case 'too_many_wrong_codes':
        return 'Too many wrong codes. The link is closed.'
      case 'wrong_code':
        return 'The code was wrong.'
      case 'cancelled':
        return 'The link was cancelled.'
      case 'expired':
        return 'The link ran out of time.'
      case 'stale_chain':
        return 'The account changed while linking. Try again.'
      default:
        return 'The link failed.'
    }
  },
} as const
