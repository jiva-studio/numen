/**
 * The rules over design tokens that stylelint has no rule for: no `var()` takes a fallback, a stylesheet
 * declares no `--numen-` token, and every `var(--numen-…)` names a token a token file declares. A fourth
 * keeps a border on one side of a box from sharing a rule with a radius. A fifth keeps the lengths to the scale:
 * a `rem` or `em` literal may stand on a property, a custom property outside the token files holds only a
 * length the token files declare, and only a 1px hairline border is written in `px`. A sixth keeps the window
 * chrome at one size: its stylesheets write no `rem` and read neither scale nor a token that follows the root font size.
 */
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import postcss from 'postcss'
import valueParser from 'postcss-value-parser'
import stylelint from 'stylelint'

const {
  createPlugin,
  utils: { report, ruleMessages, validateOptions },
} = stylelint

/** The prefix of the design tokens. */
const TOKEN_PREFIX = '--numen-'

/** Directories the token walk does not enter: installed packages, build output, fixtures; hidden directories neither. */
const SKIPPED_DIRECTORIES = new Set(['node_modules', 'dist', 'testdata'])

/** True for a stylesheet that lies in a `tokens` directory. */
export function isTokenFile(file) {
  return file.endsWith('.css') && file.split(/[\\/]/).slice(0, -1).includes('tokens')
}

/** The stylesheets below `dir` that are token files, as absolute paths. */
function findTokenFiles(dir) {
  const found = []
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name)
    if (entry.isDirectory() && !SKIPPED_DIRECTORIES.has(entry.name) && !entry.name.startsWith('.'))
      found.push(...findTokenFiles(abs))
    else if (entry.isFile() && isTokenFile(abs)) found.push(abs)
  }
  return found
}

/** The custom properties with the token prefix that the declarations of `css` declare. */
export function readDeclaredTokens(css) {
  const declared = new Set()
  postcss.parse(css).walkDecls((decl) => {
    if (decl.prop.startsWith(TOKEN_PREFIX)) declared.add(decl.prop)
  })
  return declared
}

const declaredByRoot = new Map()

/** The tokens the token files below `root` declare, read once per root. */
function readRootTokens(root) {
  if (!declaredByRoot.has(root)) {
    const declared = new Set()
    for (const file of findTokenFiles(root)) {
      for (const token of readDeclaredTokens(fs.readFileSync(file, 'utf8'))) declared.add(token)
    }
    declaredByRoot.set(root, declared)
  }
  return declaredByRoot.get(root)
}

/** The values of the tokens the token files below `root` declare, read once per root. */
const valuesByRoot = new Map()
function readRootTokenValues(root) {
  if (!valuesByRoot.has(root)) {
    const values = new Map()
    for (const file of findTokenFiles(root)) {
      postcss.parse(fs.readFileSync(file, 'utf8')).walkDecls((decl) => {
        if (decl.prop.startsWith(TOKEN_PREFIX))
          values.set(decl.prop, [...(values.get(decl.prop) ?? []), decl.value])
      })
    }
    valuesByRoot.set(root, values)
  }
  return valuesByRoot.get(root)
}

/** The tokens whose size follows a scale: a scale itself, a token holding a `rem` length, and a token reading one of those. */
export function findScaledTokens(values) {
  const scaled = new Set(['--numen-interface-scale', '--numen-text-scale'])
  let grew = true
  while (grew) {
    grew = false
    for (const [name, list] of values) {
      if (scaled.has(name)) continue
      const follows = list.some((value) => {
        let found = false
        valueParser(value).walk((node) => {
          if (
            node.type === 'word' &&
            String(valueParser.unit(node.value).unit ?? '').toLowerCase() === 'rem'
          )
            found = true
          if (
            node.type === 'function' &&
            node.value.toLowerCase() === 'var' &&
            scaled.has(node.nodes[0]?.value)
          )
            found = true
        })
        return found
      })
      if (follows) {
        scaled.add(name)
        grew = true
      }
    }
  }
  return scaled
}

/** The `var()` calls of a declaration value, each as `{ name, hasFallback, index }`. */
function findVarCalls(value) {
  const calls = []
  valueParser(value).walk((node) => {
    if (node.type !== 'function' || node.value.toLowerCase() !== 'var') return
    const name = node.nodes.find((one) => one.type === 'word')?.value ?? ''
    const hasFallback = node.nodes.some((one) => one.type === 'div' && one.value === ',')
    calls.push({ name, hasFallback, index: node.sourceIndex })
  })
  return calls
}

const NO_FALLBACK = 'numen/no-var-fallback'

const noFallbackMessages = ruleMessages(NO_FALLBACK, {
  rejected: (name) => `var(${name}, …) takes a fallback; the property is declared where it is read`,
})

const noFallback = (primary) => (root, result) => {
  if (!validateOptions(result, NO_FALLBACK, { actual: primary })) return
  root.walkDecls((decl) => {
    for (const call of findVarCalls(decl.value)) {
      if (!call.hasFallback) continue
      report({
        ruleName: NO_FALLBACK,
        result,
        node: decl,
        message: noFallbackMessages.rejected(call.name),
        word: call.name,
      })
    }
  })
}
noFallback.ruleName = NO_FALLBACK
noFallback.messages = noFallbackMessages

const NO_TOKEN_DECLARATION = 'numen/no-token-declaration'

const noTokenDeclarationMessages = ruleMessages(NO_TOKEN_DECLARATION, {
  rejected: (name) => `${name} is a design token; tokens are declared in a token or theme file`,
})

const noTokenDeclaration = (primary) => (root, result) => {
  if (!validateOptions(result, NO_TOKEN_DECLARATION, { actual: primary })) return
  root.walkDecls((decl) => {
    if (!decl.prop.startsWith(TOKEN_PREFIX)) return
    report({
      ruleName: NO_TOKEN_DECLARATION,
      result,
      node: decl,
      message: noTokenDeclarationMessages.rejected(decl.prop),
      word: decl.prop,
    })
  })
}
noTokenDeclaration.ruleName = NO_TOKEN_DECLARATION
noTokenDeclaration.messages = noTokenDeclarationMessages

const KNOWN_TOKEN = 'numen/known-token'

const knownTokenMessages = ruleMessages(KNOWN_TOKEN, {
  rejected: (name) => `${name} is declared by no token file`,
})

const knownToken = (primary, secondary) => (root, result) => {
  const isValid = validateOptions(
    result,
    KNOWN_TOKEN,
    { actual: primary },
    {
      actual: secondary,
      possible: { root: [(value) => typeof value === 'string'] },
      optional: true,
    },
  )
  if (!isValid) return
  const declared = readRootTokens(path.resolve(secondary?.root ?? process.cwd()))
  root.walkDecls((decl) => {
    for (const { name } of findVarCalls(decl.value)) {
      if (!name.startsWith(TOKEN_PREFIX) || declared.has(name)) continue
      report({
        ruleName: KNOWN_TOKEN,
        result,
        node: decl,
        message: knownTokenMessages.rejected(name),
        word: name,
      })
    }
  })
}
knownToken.ruleName = KNOWN_TOKEN
knownToken.messages = knownTokenMessages

const NO_ROUNDED_SIDE_BORDER = 'numen/no-rounded-side-border'

const noRoundedSideBorderMessages = ruleMessages(NO_ROUNDED_SIDE_BORDER, {
  rejected: (prop) =>
    `${prop} on a rounded box bends with the corners; draw the line with the row-divider class, or put it on a box without a radius`,
})

/** A border on one side or one axis of a box. */
const SIDE_BORDER =
  /^border-(top|bottom|left|right|block|block-start|block-end|inline|inline-start|inline-end)$/

/** A radius on a box: `border-radius` and every corner's own. */
const RADIUS = /^border(-(top|bottom|start|end)-(left|right|start|end))?-radius$/

/** True when the value draws nothing: `none`, `0`, `hidden`, `0px`. */
function isNoValue(value) {
  return /^(none|hidden|initial|unset|0[a-z%]*)$/i.test(value.trim())
}

/** True when a Tailwind `@apply` list names a rounded corner: any `rounded` utility but `rounded-none`. */
function appliesRadius(params) {
  return params.split(/\s+/).some((one) => /^(?:[a-z-]+:)*rounded(-(?!none$).*)?$/.test(one))
}

/** True when a Tailwind `@apply` list names a border on one side or axis. */
function appliesSideBorder(params) {
  return params
    .split(/\s+/)
    .some(
      (one) =>
        /^(?:[a-z-]+:)*border-(t|b|l|r|x|y|s|e)(-|$)/.test(one) &&
        !/border-[a-z]-(0|none)$/.test(one),
    )
}

const noRoundedSideBorder = (primary) => (root, result) => {
  if (!validateOptions(result, NO_ROUNDED_SIDE_BORDER, { actual: primary })) return
  root.walkRules((rule) => {
    const borders = []
    let rounded = false
    rule.each((node) => {
      if (node.type === 'decl') {
        if (SIDE_BORDER.test(node.prop) && !isNoValue(node.value)) borders.push(node)
        if (RADIUS.test(node.prop) && !isNoValue(node.value)) rounded = true
      } else if (node.type === 'atrule' && node.name === 'apply') {
        if (appliesRadius(node.params)) rounded = true
        if (appliesSideBorder(node.params)) borders.push(node)
      }
    })
    if (!rounded) return
    for (const node of borders) {
      const prop = node.type === 'decl' ? node.prop : node.params
      report({
        ruleName: NO_ROUNDED_SIDE_BORDER,
        result,
        node,
        message: noRoundedSideBorderMessages.rejected(prop),
      })
    }
  })
}
noRoundedSideBorder.ruleName = NO_ROUNDED_SIDE_BORDER
noRoundedSideBorder.messages = noRoundedSideBorderMessages

const NO_BARE_LENGTH = 'numen/no-bare-length'

const noBareLengthMessages = ruleMessages(NO_BARE_LENGTH, {
  pixelProperty: (length, prop) =>
    `${length} on ${prop}: a size is not written in px; write it in rem or em (only a 1px border is px)`,
  pixelDeclaration: (length, prop) =>
    `${length} in ${prop}: a custom property holds a size in rem or em; px stays out (only a 1px border is px)`,
  otherUnit: (length, prop) => `${length} on ${prop}: write the length in rem or em`,
})

/** The units of a length that does not follow the interface scale by itself: absolute and font-relative ones. Viewport and container units, percentages and `fr` are proportions and pass. */
const LENGTH_UNITS = new Set([
  'px',
  'rem',
  'em',
  'ex',
  'ch',
  'cap',
  'ic',
  'lh',
  'rlh',
  'cm',
  'mm',
  'q',
  'in',
  'pt',
  'pc',
])

/** The units that follow the root font size, written on a property or in a custom property. */
const SCALED_UNITS = new Set(['rem', 'em'])

/** The properties that draw a border line, whose width of exactly 1px is a hairline. */
const BORDER_LINE = /^border(-(top|right|bottom|left|block|inline)(-(start|end))?)?(-width)?$/

/** The lengths of a declaration value, each as `{ text, number, unit }`; zero and the contents of `url()` are not lengths. */
function findLengths(value) {
  const lengths = []
  valueParser(value).walk((node) => {
    if (node.type === 'function' && node.value.toLowerCase() === 'url') return false
    if (node.type !== 'word') return undefined
    const parsed = valueParser.unit(node.value)
    if (!parsed || !LENGTH_UNITS.has(parsed.unit.toLowerCase()) || Number(parsed.number) === 0)
      return undefined
    lengths.push({
      text: node.value,
      number: Number(parsed.number),
      unit: parsed.unit.toLowerCase(),
    })
    return undefined
  })
  return lengths
}

const noBareLength = (primary) => (root, result) => {
  if (!validateOptions(result, NO_BARE_LENGTH, { actual: primary })) return
  root.walkDecls((decl) => {
    const isCustomProperty = decl.prop.startsWith('--')
    for (const length of findLengths(decl.value)) {
      let message
      if (SCALED_UNITS.has(length.unit)) {
        continue
      } else if (isCustomProperty) {
        message =
          length.unit === 'px'
            ? noBareLengthMessages.pixelDeclaration(length.text, decl.prop)
            : noBareLengthMessages.otherUnit(length.text, decl.prop)
      } else if (length.unit === 'px' && length.number === 1 && BORDER_LINE.test(decl.prop)) {
        continue
      } else if (length.unit === 'px') {
        message = noBareLengthMessages.pixelProperty(length.text, decl.prop)
      } else {
        message = noBareLengthMessages.otherUnit(length.text, decl.prop)
      }
      report({ ruleName: NO_BARE_LENGTH, result, node: decl, message, word: length.text })
    }
  })
}
noBareLength.ruleName = NO_BARE_LENGTH
noBareLength.messages = noBareLengthMessages

const CHROME_FIXED_SIZE = 'numen/chrome-fixed-size'

const chromeFixedSizeMessages = ruleMessages(CHROME_FIXED_SIZE, {
  rem: (length, prop) =>
    `${length} in ${prop}: the window chrome keeps its size at every interface scale; write it with --numen-chrome-unit`,
  token: (name, prop) =>
    `${name} in ${prop} follows the interface scale; the window chrome reads a --numen-chrome- token`,
})

const chromePlugins = new Map()

/** True when the plugin the file belongs to draws a view in a slot of one of the bars, or draws a bar itself. */
function isChromePlugin(file, bars) {
  const dir = /^(.*[\\/]plugins[\\/][^\\/]+)[\\/]/.exec(file)?.[1]
  if (!dir) return false
  const key = `${dir}|${bars.join(',')}`
  if (!chromePlugins.has(key)) {
    let slots = []
    try {
      slots =
        JSON.parse(fs.readFileSync(path.join(dir, 'manifest.json'), 'utf8')).contributes?.slots ??
        []
    } catch {
      slots = []
    }
    chromePlugins.set(
      key,
      slots.some(
        (one) =>
          one.view && bars.some((bar) => one.slot === bar || one.slot?.startsWith(`${bar}.`)),
      ),
    )
  }
  return chromePlugins.get(key)
}

const POPOVER_CLASS = /\.[\w-]*popover[\w-]*/

/** True when every selector of the rule, or of a rule it is nested in, names a class containing "popover": the popup a widget opens. */
function isInPopover(decl) {
  for (let node = decl.parent; node; node = node.parent) {
    if (
      node.type === 'rule' &&
      node.selectors?.length &&
      node.selectors.every((selector) => POPOVER_CLASS.test(selector))
    )
      return true
  }
  return false
}

const chromeFixedSize = (primary, secondary) => (root, result) => {
  const isValid = validateOptions(
    result,
    CHROME_FIXED_SIZE,
    { actual: primary },
    {
      actual: secondary,
      possible: {
        root: [(value) => typeof value === 'string'],
        slots: [(value) => typeof value === 'string'],
      },
      optional: true,
    },
  )
  if (!isValid) return
  if (!isChromePlugin(result.opts?.from ?? root.source?.input.file ?? '', secondary?.slots ?? []))
    return
  const scaled = findScaledTokens(
    readRootTokenValues(path.resolve(secondary?.root ?? process.cwd())),
  )
  root.walkDecls((decl) => {
    if (isInPopover(decl)) return
    for (const length of findLengths(decl.value)) {
      if (length.unit !== 'rem') continue
      report({
        ruleName: CHROME_FIXED_SIZE,
        result,
        node: decl,
        message: chromeFixedSizeMessages.rem(length.text, decl.prop),
        word: length.text,
      })
    }
    for (const { name } of findVarCalls(decl.value)) {
      if (!scaled.has(name)) continue
      report({
        ruleName: CHROME_FIXED_SIZE,
        result,
        node: decl,
        message: chromeFixedSizeMessages.token(name, decl.prop),
        word: name,
      })
    }
  })
}

chromeFixedSize.ruleName = CHROME_FIXED_SIZE
chromeFixedSize.messages = chromeFixedSizeMessages

export default [
  createPlugin(NO_FALLBACK, noFallback),
  createPlugin(NO_TOKEN_DECLARATION, noTokenDeclaration),
  createPlugin(KNOWN_TOKEN, knownToken),
  createPlugin(NO_ROUNDED_SIDE_BORDER, noRoundedSideBorder),
  createPlugin(NO_BARE_LENGTH, noBareLength),
  createPlugin(CHROME_FIXED_SIZE, chromeFixedSize),
]
