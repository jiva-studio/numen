/**
 * The test identities: one person with recovery words and three computers that hold the keys of the account, and the
 * state of the fake sync server that knows that account. The files are made by `npm run acceptance:identity` from the
 * code of the application, so they are regenerated when the format of the keys changes.
 */
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const FOLDER = path.resolve(here, '../assets/identities')
const SYNCSERVER = path.resolve(here, '../../../../tools/syncserver')
const GENERATOR = path.join(SYNCSERVER, 'bin', 'seedidentity')
const IDENTITY_FILE = 'identity.json'
const SERVER_FILE = 'server.json'
export const OUT_OF_DATE = 'the test identities are out of date: run npm run acceptance:identity'

export type IdentityComputer = 'first' | 'second' | 'third'

interface SeedFile {
  readonly words: readonly string[]
  readonly fingerprint: string
  readonly chain: string
  readonly computers: Readonly<Record<IdentityComputer, { readonly credentials: Readonly<Record<string, string>> }>>
}

/** The setting of the application that says its keys are kept as plaintext in the credentials file, which is what a run with no keychain needs. */
export const KEY_FILE_SETTINGS = { app: { keys: { credentialsFile: 'plaintext' } } } as const

let loaded: SeedFile | undefined

const readSeed = (): SeedFile => {
  if (loaded !== undefined) return loaded
  let text: string
  try {
    text = fs.readFileSync(path.join(FOLDER, IDENTITY_FILE), 'utf8')
  } catch {
    throw new Error(OUT_OF_DATE)
  }
  loaded = JSON.parse(text) as SeedFile
  return loaded
}

export const getSeedWords = (): readonly string[] => readSeed().words

export const getSeedFingerprint = (): string => readSeed().fingerprint

/** Puts the state of the fake sync server that knows the account into its state folder. */
export const writeSeedServerState = (stateFolder: string): void => {
  fs.copyFileSync(path.join(FOLDER, SERVER_FILE), path.join(stateFolder, SERVER_FILE))
}

/** Puts the keys of a computer where the application reads them at its start: the credentials file and the chain file next to the settings. */
export const writeSeedKeys = (settingsFolder: string, computer: IdentityComputer): void => {
  const seed = readSeed()
  const credentials = path.join(settingsFolder, 'credentials')
  fs.mkdirSync(credentials, { recursive: true, mode: 0o700 })
  fs.writeFileSync(path.join(credentials, 'credentials.json'), JSON.stringify(seed.computers[computer].credentials), { mode: 0o600 })
  const identity = path.join(settingsFolder, 'identity')
  fs.mkdirSync(identity, { recursive: true, mode: 0o700 })
  fs.writeFileSync(path.join(identity, 'chain.bin'), Buffer.from(seed.chain, 'base64'), { mode: 0o600 })
}

/** Fails with a plain message when the saved identities are not what the code of the application makes now. */
export const checkSeedIsCurrent = (): void => {
  if (!fs.existsSync(GENERATOR)) {
    throw new Error(`${GENERATOR} is missing; build it with make app_desktop_acceptance`)
  }
  try {
    execFileSync(GENERATOR, ['-check', '-out', FOLDER], { stdio: 'pipe' })
  } catch (error) {
    const detail = String((error as { stderr?: Buffer }).stderr ?? '').trim()
    throw new Error(`${OUT_OF_DATE}${detail === '' ? '' : `\n${detail}`}`, { cause: error })
  }
}
