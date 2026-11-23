/** The answers of the person to a link in progress: the new device, the code and the account. */
import type { Result } from '@numen/result'
import type { LinkCalls } from '../lib/types'

type Ask = (call: () => Promise<Result<unknown, string>>) => Promise<void>

export function createLinkAnswers(getCalls: () => LinkCalls, ask: Ask) {
  return {
    decide: (isApproved: boolean) => ask(() => getCalls().decide(isApproved)),
    sendCode: (code: string) => ask(() => getCalls().enterCode(code)),
    confirmJoin: (isApproved: boolean) => ask(() => getCalls().confirmJoin(isApproved)),
  }
}
