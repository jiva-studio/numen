/**
 * The environment of the application: the things it reaches outside the window, which a scenario sets
 * and the stand-ins of the server of the world answer for. The rules here need no browser.
 */
import fs from 'node:fs'
import type { JsonObject, JsonValue } from '../../../e2e/desktop/tests/support/json.js'

type EnvironmentName = 'agent' | 'embedding model' | 'latest release'

export interface EnvironmentSetting {
  readonly name: EnvironmentName
  readonly state: string
}

const NAMES: readonly EnvironmentName[] = ['agent', 'embedding model', 'latest release']
const SPEEDS = ['slow', 'normal']
const VERSION = /^\d+\.\d+\.\d+$/

const isName = (text: string): text is EnvironmentName => (NAMES as readonly string[]).includes(text)

/** Reads the rows `name | state` of the table of `the environment is set to the following:`. */
export const parseEnvironment = (rows: readonly (readonly string[])[]): EnvironmentSetting[] => {
  if (rows.length === 0) throw new Error(`the environment table is empty: write rows like "| agent | slow |"`)
  return rows.map((row) => {
    if (row.length !== 2) throw new Error(`the row "${row.join(' | ')}" must have a name and a state: "| agent | slow |"`)
    const [name = '', state = ''] = row
    if (!isName(name)) throw new Error(`"${name}" is not part of the environment: use ${NAMES.join(', ')}`)
    if (name === 'latest release') {
      if (state !== 'none' && !VERSION.test(state)) {
        throw new Error(`"${state}" is not a state of the latest release: use none or a version like 2.0.0`)
      }
    } else if (!SPEEDS.includes(state)) {
      throw new Error(`"${state}" is not a state of the ${name}: use ${SPEEDS.join(' or ')}`)
    }
    return { name, state }
  })
}

/** The build number the stand-in release gets for a version: a later version has a later build. */
export const getReleaseBuild = (version: string): number => {
  const [major = 0, minor = 0, patch = 0] = version.split('.').map(Number)
  return major * 1_000_000 + minor * 1_000 + patch
}

/** What the stand-in of the release feed names as the latest release. */
export const createRelease = (version: string): JsonObject => ({
  version,
  build: String(getReleaseBuild(version)),
  channel: 'stable',
  released: '2026-10-30T00:00:00Z',
  files: [],
})

export interface EnvironmentTargets {
  /** The address of the stand-ins. */
  readonly origin: string
  /** The file the mock embedding model holds its batches while it exists. */
  readonly embeddingHoldFile: string
}

const send = async (method: string, url: string, body?: JsonValue): Promise<void> => {
  const answer = await fetch(url, {
    method,
    ...(body === undefined ? {} : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  })
  if (!answer.ok) throw new Error(`the stand-in answered ${answer.status} to ${method} ${url}`)
}

/** Puts the settings in force in the stand-ins and the files they are read from. */
export const applyEnvironment = async (settings: readonly EnvironmentSetting[], targets: EnvironmentTargets): Promise<void> => {
  for (const { name, state } of settings) {
    if (name === 'agent') {
      if (state === 'slow') await send('PUT', `${targets.origin}/stand-in/hold`, { all: true })
      else await send('DELETE', `${targets.origin}/stand-in/hold`)
    } else if (name === 'embedding model') {
      if (state === 'slow') fs.writeFileSync(targets.embeddingHoldFile, '')
      else fs.rmSync(targets.embeddingHoldFile, { force: true })
    } else if (state === 'none') {
      await send('DELETE', `${targets.origin}/stand-in/release`)
    } else {
      await send('PUT', `${targets.origin}/stand-in/release`, createRelease(state))
    }
  }
}
