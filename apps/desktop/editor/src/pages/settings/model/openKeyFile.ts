import type { KeyFileStore } from '@/entities/key-file'

/** Reads how the keys are kept and asks for the passphrase when the key file is locked. */
export async function openKeyFile(keyFile: Pick<KeyFileStore, 'load' | 'askWhenNeeded'>): Promise<void> {
  const found = await keyFile.load()
  if (found?.isLocked) await keyFile.askWhenNeeded()
}
