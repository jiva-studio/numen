/** The option the person picked while it is still offered, and the first option otherwise; empty where none is offered. */
export function getChoice(picked: string, options: readonly string[]): string {
  return options.includes(picked) ? picked : (options[0] ?? '')
}
