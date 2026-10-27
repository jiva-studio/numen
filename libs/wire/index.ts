/**
 * What the windows' ports are answered with over the wire.
 *
 * `@numen/plugin-sdk` declares the ports and never sees the protocol; a window draws and
 * never translates. What turns one into the other stands here, once.
 */
export { createAgentPort } from './agent/agent'
export type { AgentClient } from './agent/agent'
export { getNames } from './settings/enumNames'
export { transport } from './transport'
export { formatErrorMessage, formatErrorCodeMessage } from './errors/error'
export { asFailure, asValue } from './result'
export type { Failure, Result } from './result'
export {
  ERROR_CODE,
  findErrorCode,
  formatFingerprint,
  getAssetUrl,
  getBytesQuery,
  isStaleAnswer,
  MAX_BUSY_RETRIES,
  parseFingerprint,
  retryWhileBusy,
} from './errors/answers'
export type { ErrorCode } from './errors/errorCode'
export { ERRORS, getErrorWords, getKnownErrorCode } from './errors/codes'
