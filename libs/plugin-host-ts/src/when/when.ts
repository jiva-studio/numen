/**
 * The when conditions of commands and slot entries: CEL expressions over the variables of
 * libs/protocol/when.json, such as `target.isFolder && !target.isMultiple`. The Go reader of the
 * manifest checks them, types included; the window reads a condition once and evaluates it over the
 * values of each place.
 */
import { celEnv, isCelError, mapType, parse, plan, CelScalar, type CelType } from '@bufbuild/cel'
import { asFailure, asValue, WHEN_VARIABLES, type Result } from '@numen/plugin-sdk'

/** The settings by their dotted keys, nested: config.editor.lineNumbers reads `{ editor: { lineNumbers } }`. */
export interface WhenSettings {
  readonly [key: string]: unknown
}

/**
 * The values of a place, by the qualified names of the variables, such as `target.isNote`, and
 * `config` for the settings. A variable the values do not give is false, or empty for a string.
 */
export type WhenValues = Readonly<Record<string, unknown>> & { readonly config?: WhenSettings }

/** A condition read once and evaluated over the values of many places. */
export interface WhenCondition {
  /** The variables it reads, sorted; the settings count as `config`. */
  readonly names: readonly string[]
  evaluate(values: WhenValues): boolean
}

const TYPES: Readonly<Record<string, CelType>> = {
  bool: CelScalar.BOOL,
  string: CelScalar.STRING,
  map: mapType(CelScalar.STRING, CelScalar.DYN),
}

const ENVIRONMENT = celEnv({
  variables: Object.fromEntries(WHEN_VARIABLES.map((one) => [one.name, TYPES[one.type]!])),
})

const DECLARED = new Set<string>(WHEN_VARIABLES.map((one) => one.name))

/** The value a variable has where the values do not give it. */
const getDefault = (type: string): unknown => {
  if (type === 'bool') return false
  if (type === 'string') return ''
  return new Map()
}

type Expr = ReturnType<typeof parse>['expr'] | undefined

/** `target.isNote` for the selection of isNote on target; nothing for an expression of another kind. */
function getQualifiedName(expr: Expr): string | undefined {
  const kind = expr?.exprKind
  if (kind?.case === 'identExpr') return kind.value.name
  if (kind?.case !== 'selectExpr' || kind.value.testOnly) return undefined
  const operand = getQualifiedName(kind.value.operand)
  return operand === undefined ? undefined : `${operand}.${kind.value.field}`
}

type ExprKind = NonNullable<Expr>['exprKind']

/** The expressions an expression holds, which may read variables of their own. */
const CHILDREN: {
  readonly [K in NonNullable<ExprKind['case']>]: (
    value: Extract<ExprKind, { case: K }>['value'],
  ) => Expr[]
} = {
  constExpr: () => [],
  identExpr: () => [],
  selectExpr: (value) => [value.operand],
  callExpr: (value) => [value.target, ...value.args],
  listExpr: (value) => value.elements,
  structExpr: (value) =>
    value.entries.flatMap((one) => [
      one.keyKind.case === 'mapKey' ? one.keyKind.value : undefined,
      one.value,
    ]),
  comprehensionExpr: (value) => [value.iterRange],
}

const getChildren = (expr: Expr): Expr[] => {
  const kind = expr?.exprKind
  if (!kind?.case) return []
  return (CHILDREN[kind.case] as (value: unknown) => Expr[])(kind.value)
}

/** The variables the expression reads, each as declared, the settings as `config`. */
function collectNames(expr: Expr, names: Set<string>): void {
  const qualified = getQualifiedName(expr)
  if (qualified === undefined) {
    for (const one of getChildren(expr)) collectNames(one, names)
  } else {
    names.add(qualified === 'config' || qualified.startsWith('config.') ? 'config' : qualified)
  }
}

/** A plain object of settings as the map CEL selects keys of. */
function toCelMap(value: unknown): unknown {
  if (value === null || typeof value !== 'object' || Array.isArray(value) || value instanceof Map) {
    return value
  }
  return new Map(Object.entries(value).map(([key, one]) => [key, toCelMap(one)]))
}

const ALWAYS: WhenCondition = { names: [], evaluate: () => true }

/** Reads a condition; a failure where it is malformed or reads a variable that is not declared. */
export function parseWhen(text: string): Result<WhenCondition, string> {
  if (text === '') return asValue(ALWAYS)
  let parsed: ReturnType<typeof parse>
  try {
    parsed = parse(text)
  } catch (error) {
    return asFailure(`"${text}": ${(error as Error).message}`)
  }
  const found = new Set<string>()
  collectNames(parsed.expr, found)
  const unknown = [...found].filter((name) => !DECLARED.has(name))
  if (unknown.length > 0)
    return asFailure(`"${text}" reads ${unknown.join(', ')}, which is no variable of when`)
  const names = [...found].sort()
  const run = plan(ENVIRONMENT, parsed)
  return asValue({
    names,
    evaluate: (values) => {
      const bindings = Object.fromEntries(
        WHEN_VARIABLES.map((one) => [one.name, toCelMap(values[one.name] ?? getDefault(one.type))]),
      )
      const result = run(bindings as Parameters<typeof run>[0])
      return !isCelError(result) && result === true
    },
  })
}
