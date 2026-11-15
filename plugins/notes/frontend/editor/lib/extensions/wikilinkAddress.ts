/** A scheme is letters, digits, `+`, `-` and `.`, written before the `://`. */
const SCHEMED = /^([A-Za-z0-9+\-.]+):\/\/([\s\S]*)$/

/** The address between brackets as `scheme://value`; empty for no target. */
export const getAddress = (inside: string): string => {
  let target = inside
  const alias = target.indexOf('|')
  if (alias >= 0) target = target.slice(0, alias)
  const fragment = target.indexOf('#')
  if (fragment >= 0) target = target.slice(0, fragment)
  target = target.trim()
  const schemed = SCHEMED.exec(target)
  const [scheme, value] = schemed ? [schemed[1], schemed[2]] : ['name', target]
  return value ? `${scheme}://${value}` : ''
}
