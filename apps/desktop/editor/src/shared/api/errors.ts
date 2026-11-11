/**
 * The error text and codes the window shows. Slices reach the core client's error helpers through
 * this file.
 */
import { formatErrorMessage } from '@numen/core-client'

/** Whatever a try block threw or a promise rejected with. */
export type { Caught, Unvalidated } from '@numen/core-client'

/** The error code a vault operation answers with. */
export type { ErrorCode } from '@numen/plugin-loader'

/** The sentence an error is reported as. */
export const formatError = formatErrorMessage

/** An error whose message is shown as it is. */
export { ReportedError } from '@numen/core-client'

/** The sentence of each error a vault operation answers with. */
export { ERRORS } from '@numen/core-client'
