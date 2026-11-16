/**
 * What the calls of the SDK share: the file an answer came out of, and the address a file is
 * requested at.
 *
 * A file is one value a caller carries about and never reads into. The schema
 * holds the parts; what a tab does with one is present it back unchanged, so
 * the parts stay here and the string goes everywhere else.
 */
import { Code, type ConnectError } from '@connectrpc/connect'

/**
 * The file an answer came out of: where it is, how large it was, when it changed and, where the answer
 * read the content, its hash and the version of the vault.
 */
type FileStat = {
  path: string
  size: bigint
  mtime: bigint
  /** The SHA-256 of the content in lowercase hex; empty where the answer did not read it. */
  hash?: string
  /** The number of writes the vault had taken when the answer was made; zero where it does not say. */
  version?: bigint
}

/**
 * The file an answer came out of, as the one string the window carries: `<size>:<version>:<hash>
 * <mtime> <path>`. The hash is empty where the answer read no content and the version is 0 where it
 * does not say.
 */
export const formatFingerprint = (at?: FileStat): string | undefined => {
  if (!at) return undefined
  return `${at.size}:${at.version ?? 0n}:${at.hash ?? ''} ${at.mtime} ${at.path}`
}

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const parseFingerprint = (stat: string) => {
  // Stryker disable next-line StringLiteral: BigInt evaluates empty string and '0' both to 0n.
  const [head = '0', mtime = '0', ...rest] = stat.split(' ')
  const [size = '0', version = '0', hash = ''] = head.split(':')
  return {
    path: rest.join(' '),
    size: BigInt(size),
    mtime: BigInt(mtime),
    hash,
    version: BigInt(version),
  }
}

/**
 * The address of a vault file query. The path is written in full, so a file in
 * a folder is one part of the address and the queried facet is the next.
 */
export const getAssetUrl = (path: string): string => `/assets/${encodeURIComponent(path)}`

/**
 * Which bytes an address refers to, as the address spells them. The path is
 * already part of the address, so this holds the remaining parts that identify
 * the file.
 */
export const getBytesQuery = (file: string): string => {
  const stat = parseFingerprint(file)
  return `size=${stat.size}&mtime=${stat.mtime}`
}

/** How often a document that is busy is waited out before it errors. */
const MAX_BUSY_RETRIES = 3

/** How long the window waits before asking a busy document again. */
const BUSY_RETRY_DELAY_MS = 1000

/**
 * Retries a call while the application reports that the document is busy: a
 * document held by a client that is rendering from it is requested again after
 * a delay.
 */
export const retryWhileBusy = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let attempt = 0; ; attempt++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || attempt >= MAX_BUSY_RETRIES)
        throw error
      await sleep(BUSY_RETRY_DELAY_MS)
    }
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
