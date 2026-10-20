/**
 * What the windows' ports are answered with over the wire.
 *
 * `libs/ui` declares the ports and never sees the protocol; a window draws and
 * never translates. What turns one into the other stands here, once.
 */
export { createAgentPort } from './agent'
export { goalNames, goalOf } from './goal'
export type { Goal } from './goal'
export { getNames } from './naming'
export { transport } from './transport'
export { formatErrorMessage, formatErrorCodeMessage } from './error'
export { asFailure, asValue } from './result'
export type { Failure, Result } from './result'
export { DEFAULT_SETTINGS, formatSettings, parseSettings } from './settings'
export type { BudgetUnit, Load, Rule, Settings, SettingsMessage } from './settings'
export { formatStopReasonMessage } from './stop'
export type { StopDetail } from './stop'
export {
  ERROR_CODE,
  findErrorCode,
  formatFingerprint,
  getAssetUrl,
  getBytesQuery,
  isStaleAnswer,
  parseFingerprint,
  retryWhileBusy,
} from './answers'
export type { ErrorCode } from './errorCode'
export { ERRORS, getErrorWords, getKnownErrorCode } from './codes'
