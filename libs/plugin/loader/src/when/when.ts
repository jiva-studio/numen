/**
 * The when conditions of commands and slot entries: CEL expressions over the variables of
 * libs/core/protocol/when.json, such as `target.isFolder && !target.isMultiple`. The Go reader of the
 * manifest checks them, types included; the window reads a condition once and evaluates it over the
 * values of each place.
 */
import {
  celEnv,
  celError,
  isCelError,
  listType,
  mapType,
  parse,
  plan,
  CelScalar,
  type CelType,
} from '@bufbuild/cel'
import {
  asFailure,
  asValue,
  WHEN_VARIABLES,
  type JsonObject,
  type JsonValue,
  type Result,
} from '@numen/plugin-sdk'

/** The settings by their dotted keys, nested: config.editor.lineNumbers reads `{ editor: { lineNumbers } }`. */
export type WhenSettings = JsonObject

/**
 * The values of a place, by the qualified names of the variables, such as `target.isNote`, and
 * `config` for the settings. A variable the values do not give is false, or empty for a string.
 */
export type WhenValues = Readonly<Record<string, JsonValue | undefined>> & {
  readonly config?: WhenSettings
}

/** The value of a condition over values that may leave variables out: unknown where its value depends on one of them. */
type WhenResult = 'true' | 'false' | 'unknown'

/** A condition read once and evaluated over the values of many places. */
export interface WhenCondition {
  readonly names: readonly string[]
  evaluate(values: WhenValues): boolean
  /**
   * The value over the values, a variable they do not give being unknown. A condition that fails to
   * evaluate for another reason, such as a setting config does not hold, is unknown as well.
   */
  evaluatePartial(values: WhenValues): WhenResult
}

const TYPES: Readonly<Record<(typeof WHEN_VARIABLES)[number]['type'], CelType>> = {
  bool: CelScalar.BOOL,
  string: CelScalar.STRING,
  map: mapType(CelScalar.STRING, CelScalar.DYN),
  list: listType(mapType(CelScalar.STRING, CelScalar.DYN)),
}

const ENVIRONMENT = celEnv({
  variables: Object.fromEntries(WHEN_VARIABLES.map((one) => [one.name, TYPES[one.type]])),
})

const DECLARED = new Set<string>(WHEN_VARIABLES.map((one) => one.name))

const getDefaultValue = (type: string): JsonValue => {
  if (type === 'bool') return false
  if (type === 'string') return ''
  if (type === 'list') return []
  return {}
}

type Expr = ReturnType<typeof parse>['expr'] | undefined

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

type ReadChildren = (value: NonNullable<ExprKind['value']>) => Expr[]

const getChildren = (expr: Expr): Expr[] => {
  const kind = expr?.exprKind
  if (!kind?.case) return []
  return (CHILDREN[kind.case] as ReadChildren)(kind.value)
}

/** The variables of the type map, whose keys a condition selects: `config` and `target.properties`. */
const MAPS: readonly string[] = WHEN_VARIABLES.filter((one) => one.type === 'map').map(
  (one) => one.name,
)

const getVariableName = (qualified: string): string =>
  MAPS.find((one) => qualified === one || qualified.startsWith(one + '.')) ?? qualified

function collectNames(expr: Expr, names: Set<string>): void {
  const qualified = getQualifiedName(expr)
  if (qualified === undefined) {
    for (const one of getChildren(expr)) collectNames(one, names)
  } else {
    names.add(getVariableName(qualified))
  }
}

/** A value as the evaluator reads it: a JSON object is a map. */
type CelValue =
  string | number | boolean | null | readonly CelValue[] | ReadonlyMap<string, CelValue>

function toCelMap(value: JsonValue): CelValue {
  if (Array.isArray(value)) return value.map(toCelMap)
  if (value === null || typeof value !== 'object') return value
  return new Map(Object.entries(value).map(([key, one]) => [key, toCelMap(one)]))
}

/** The value of a variable that is not given: an error, which `&&` and `||` absorb where the other side decides. */
const UNKNOWN = celError('the value of the variable is not known')

const ALWAYS: WhenCondition = { names: [], evaluate: () => true, evaluatePartial: () => 'true' }

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
  const undeclared = [...found].filter((name) => !DECLARED.has(name))
  if (undeclared.length > 0)
    return asFailure(`"${text}" reads ${undeclared.join(', ')}, which is no variable of when`)
  const names = [...found].sort()
  const run = plan(ENVIRONMENT, parsed)
  return asValue({
    names,
    evaluate: (values) => {
      const bindings = Object.fromEntries(
        WHEN_VARIABLES.map((one) => [one.name, toCelMap(values[one.name] ?? getDefaultValue(one.type))]),
      )
      const result = run(bindings)
      return !isCelError(result) && result === true
    },
    evaluatePartial: (values) => {
      const bindings = Object.fromEntries(
        WHEN_VARIABLES.map((one) => {
          const value = values[one.name]
          return [one.name, value === undefined ? UNKNOWN : toCelMap(value)]
        }),
      )
      const result = run(bindings as Parameters<typeof run>[0])
      if (isCelError(result)) return 'unknown'
      return result === true ? 'true' : 'false'
    },
  })
}
