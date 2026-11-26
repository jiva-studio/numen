/**
 * The identity of one scenario: one person with recovery words and three computers that hold the keys of the account. The Go tool
 * `syncaccount seed` makes it on the real sync server for each scenario, from a label that no other scenario has, so scenarios that run side
 * by side never share an account.
 */
import { randomUUID } from 'node:crypto'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { holdForRemoval, freeFromRemoval } from './reaper.js'
import { seedAccount, type SyncStack } from './sync-stack.js'

const IDENTITY_FILE = 'identity.json'

export type IdentityComputer = 'first' | 'second' | 'third'

export interface IdentitySeed {
  readonly words: readonly string[]
  readonly fingerprint: string
  readonly chain: string
  readonly computers: Readonly<Record<IdentityComputer, { readonly credentials: Readonly<Record<string, string>> }>>
}

/** The setting of the application that says its keys are kept as plaintext in the credentials file, which is what a run with no keychain needs. */
export const KEY_FILE_SETTINGS = { app: { keys: { credentialsFile: 'plaintext' } } } as const

/** The text of a label no other scenario and no other run has: the real server keeps every account it has seen. */
export const createLabel = (): string => `acceptance ${process.pid} ${randomUUID()}`

/** Makes the account of a new label on the server and returns its seed; the folder it was written into is removed with `removeSeed`. */
export const makeSeed = async (stack: SyncStack, plan: string | undefined): Promise<{ readonly seed: IdentitySeed; readonly folder: string }> => {
  const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-sync-account-'))
  holdForRemoval('dir', folder)
  await seedAccount(stack, createLabel(), folder, plan)
  return { seed: JSON.parse(fs.readFileSync(path.join(folder, IDENTITY_FILE), 'utf8')) as IdentitySeed, folder }
}

export const removeSeed = (folder: string): void => {
  fs.rmSync(folder, { recursive: true, force: true })
  freeFromRemoval('dir', folder)
}

/** Puts the keys of a computer where the application reads them at its start: the credentials file and the chain file next to the settings. */
export const writeSeedKeys = (settingsFolder: string, computer: IdentityComputer, seed: IdentitySeed): void => {
  const credentials = path.join(settingsFolder, 'credentials')
  fs.mkdirSync(credentials, { recursive: true, mode: 0o700 })
  fs.writeFileSync(path.join(credentials, 'credentials.json'), JSON.stringify(seed.computers[computer].credentials), { mode: 0o600 })
  const identity = path.join(settingsFolder, 'identity')
  fs.mkdirSync(identity, { recursive: true, mode: 0o700 })
  fs.writeFileSync(path.join(identity, 'chain.bin'), Buffer.from(seed.chain, 'base64'), { mode: 0o600 })
}
