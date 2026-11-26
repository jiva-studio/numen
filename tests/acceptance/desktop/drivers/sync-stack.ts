/**
 * The real sync server the scenarios run against, the stack of tests/server or the stand of the owner: where it is, how a scenario makes an
 * account on it, pays for a plan at its payment stand-in and reads what it stores. The harness starts nothing; the address, the root key of
 * the epochs and the stand-in come from the environment the Go tests of Numen Sync read as well.
 */
import { execFile } from 'node:child_process'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'

export const ENV_URL = 'NUMEN_SYNC_TEST_URL'
export const ENV_ROOT_KEY = 'NUMEN_SYNC_TEST_ROOT_KEY'
export const ENV_PAYMENT_URL = 'NUMEN_SYNC_TEST_PAYMENT_URL'
export const ENV_BUCKET_URL = 'NUMEN_SYNC_TEST_BUCKET_URL'
export const ENV_REQUIRED = 'NUMEN_SYNC_TEST_REQUIRED'
/** The tag of a feature or scenario that runs against the real sync server. */
export const SYNC_SERVER_TAG = '@sync-server'
/** The tag of a scenario that needs a power over the server that the shared stack does not give: its process, its clock, its data or the network of one computer. */
export const OWN_SERVER_TAG = '@own-server'
export const NO_STACK_MESSAGE = `no real sync server: run make sync_stack_up, then eval "$(make -s sync_stack_env)" to set ${ENV_URL} and ${ENV_ROOT_KEY}`

const ROOT_KEY_PATTERN = /^[0-9a-f]{64}$/

const TOOL = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../tools/syncaccount/bin/syncaccount')

const run = promisify(execFile)

export interface SyncStack {
  readonly url: string
  readonly rootKey: string
  readonly paymentUrl: string | undefined
  readonly bucketUrl: string | undefined
}

type Environment = Readonly<Record<string, string | undefined>>

/** The stack the environment names, or undefined when it names none; a root key that is not 64 hex digits is an error. */
export const readStack = (env: Environment = process.env): SyncStack | undefined => {
  const url = env[ENV_URL]
  const rootKey = env[ENV_ROOT_KEY]
  if (url === undefined || url === '' || rootKey === undefined || rootKey === '') return undefined
  if (!ROOT_KEY_PATTERN.test(rootKey)) throw new Error(`${ENV_ROOT_KEY} is not 64 hex digits of a public key`)
  return { url: url.replace(/\/+$/, ''), rootKey, paymentUrl: trimAddress(env[ENV_PAYMENT_URL]), bucketUrl: trimAddress(env[ENV_BUCKET_URL]) }
}

const trimAddress = (text: string | undefined): string | undefined => (text === undefined || text === '' ? undefined : text.replace(/\/+$/, ''))

/** Whether a run without the stack fails: the job that runs the stack says so, and a job without it, such as a pull request from a fork, skips. */
export const isStackRequired = (env: Environment = process.env): boolean => env[ENV_REQUIRED] === '1'

export const requireStack = (): SyncStack => {
  const stack = readStack()
  if (stack === undefined) throw new Error(NO_STACK_MESSAGE)
  return stack
}

const requirePayment = (stack: SyncStack): string => {
  if (stack.paymentUrl === undefined) throw new Error(`no payment stand-in: set ${ENV_PAYMENT_URL} to the control API of the stack`)
  return stack.paymentUrl
}

const runTool = async (args: readonly string[]): Promise<string> => {
  try {
    return (await run(TOOL, [...args], { timeout: 150_000 })).stdout.trim()
  } catch (error) {
    const detail = String((error as { stderr?: Buffer | string }).stderr ?? '').trim()
    if ((error as { code?: string }).code === 'ENOENT') throw new Error(`${TOOL} is missing; build it with make app_desktop_acceptance`, { cause: error })
    throw new Error(`syncaccount ${args[0]} failed${detail === '' ? '' : `: ${detail}`}`, { cause: error })
  }
}

/** Makes the account of the label on the server, with three computers, and writes identity.json into the folder; with a plan, the account buys it at the stand-in. */
export const seedAccount = async (stack: SyncStack, label: string, folder: string, plan: string | undefined): Promise<void> => {
  const payment = plan === undefined ? [] : ['-plan', plan, '-payment', requirePayment(stack)]
  await runTool(['seed', '-label', label, '-server', stack.url, '-out', folder, ...payment])
}

/** The fingerprints of the devices the chain of the account of the words holds as active; none for an account the server does not know. */
export const readDevices = async (stack: SyncStack, words: readonly string[]): Promise<string[]> =>
  JSON.parse(await runTool(['list-devices', '-server', stack.url, '-words', words.join(' ')])) as string[]

/** Fails when the text is in an object the bucket received since the time, and when it received none. */
export const expectTextNotStored = async (stack: SyncStack, text: string, since: Date): Promise<void> => {
  if (stack.bucketUrl === undefined) throw new Error(`no bucket: set ${ENV_BUCKET_URL} to the S3 address of the stack`)
  await runTool(['find-text', '-bucket', stack.bucketUrl, '-text', text, '-since', String(Math.floor(since.getTime() / 1000))])
}

/** The id of the checkout in the address of its payment page. */
export const getCheckoutId = (address: string): string => {
  const found = /\/pay\/([^/?#]+)/.exec(address)
  if (found?.[1] === undefined) throw new Error(`"${address}" is not the address of a payment page`)
  return found[1]
}

/** Pays the checkout at the control API of the payment stand-in, as a person does on its page; the subscription runs for 30 days. */
export const sendPayment = async (stack: SyncStack, checkout: string): Promise<void> => {
  const periodEnd = Math.floor(Date.now() / 1000) + 30 * 24 * 3600
  const answer = await fetch(`${requirePayment(stack)}/pay?checkout=${encodeURIComponent(checkout)}&period_end=${periodEnd}`, { method: 'POST' })
  if (!answer.ok) throw new Error(`the control API of the payment stand-in answered ${answer.status} to the payment: ${await answer.text()}`)
}

/** The plans the stand-in sells, as the name and the size in GB a person reads. */
export const readPlans = async (stack: SyncStack): Promise<{ readonly name: string; readonly gigabytes: number }[]> => {
  const answer = await fetch(`${requirePayment(stack)}/products`)
  if (!answer.ok) throw new Error(`the control API of the payment stand-in answered ${answer.status} to the list of products`)
  const rows = (await answer.json()) as { name: string; quota_gib?: string }[]
  return rows.flatMap((row) => (row.quota_gib === undefined ? [] : [{ name: row.name, gigabytes: Number(row.quota_gib) }]))
}
