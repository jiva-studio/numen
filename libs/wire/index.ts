/**
 * What the windows' ports are answered with over the wire.
 *
 * `libs/ui` declares the ports and never sees the protocol; a window draws and
 * never translates. What turns one into the other stands here, once.
 */
export { createAgentPort } from './agent/agent'
export { goalNames, goalOf } from './settings/goal'
export type { Goal } from './settings/goal'
export { getNames } from './settings/enumNames'
export { transport } from './transport'
export { formatErrorMessage, formatErrorCodeMessage } from './errors/error'
export { asFailure, asValue } from './result'
export type { Failure, Result } from './result'
export { DEFAULT_SETTINGS, formatSettings, parseSettings } from './settings/settings'
export type { BudgetUnit, Load, Rule, Settings, SettingsMessage } from './settings/settings'
export { formatStopReasonMessage } from './agent/stop'
export type { StopDetail } from './agent/stop'
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
