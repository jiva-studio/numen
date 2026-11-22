/**
 * The error text the window shows. Slices reach the core client's error words through this file.
 */
import { formatErrorMessage } from '@numen/result'

/** The sentence an error is reported as. */
export const formatError = formatErrorMessage

/** The sentence of each error a vault operation answers with. */
export { ERRORS } from '@numen/core-client'
