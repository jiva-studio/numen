/**
 * The custom properties a piece of CSS sets, read the way the browser parses it: the declarations
 * of every rule, by property name.
 */
export const readTokens = (css: string): ReadonlyMap<string, string> => {
  const sheet = document.implementation.createHTMLDocument('')
  const style = sheet.createElement('style')
  style.textContent = css
  sheet.head.appendChild(style)
  const tokens = new Map<string, string>()
  for (const rule of Array.from(style.sheet?.cssRules ?? [])) {
    if (!('style' in rule)) continue
    const { style: declarations } = rule as CSSStyleRule
    for (const name of Array.from({ length: declarations.length }, (_, at) => declarations.item(at))) {
      tokens.set(name, declarations.getPropertyValue(name).trim())
    }
  }
  return tokens
}
