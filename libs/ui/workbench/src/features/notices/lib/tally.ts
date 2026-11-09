/**
 * The vocabulary a notice and a task share, as plain values.
 */

/** How a message is classified: information, a warning, or an error. */
export type Severity = 'info' | 'warning' | 'error'

/** The unit of a count. Bytes are formatted as sizes and seconds as a clock; the rest is counted one by one. */
export type TallyUnit = 'items' | 'bytes' | 'seconds'
