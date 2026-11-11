import type { CommandInvocation } from './command'

/** The values a keybinding or a menu entry runs a command with. */
export type Values = Readonly<Record<string, string>>

/** The values as a field of a run, left out where there are none. */
export const getValues = (values: Values | undefined): { values?: Values } =>
  values && Object.keys(values).length > 0 ? { values } : {}

/** The invocation with the values it is run with. */
export const withValues = (
  invocation: CommandInvocation,
  values: Values | undefined,
): CommandInvocation => ({
  ...invocation,
  ...getValues(values),
})
