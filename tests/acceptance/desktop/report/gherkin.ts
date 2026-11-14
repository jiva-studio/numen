/** The scenarios of a feature file as the author wrote them, with the values of an outline's example row put in. */
import fs from 'node:fs'
import { AstBuilder, compile, GherkinClassicTokenMatcher, Parser } from '@cucumber/gherkin'
import { IdGenerator } from '@cucumber/messages'
import type {
  FeatureChild,
  Pickle,
  PickleStep,
  PickleStepArgument,
  RuleChild,
  Step,
} from '@cucumber/messages'

export type GherkinStep = {
  readonly keyword: string
  readonly text: string
  readonly line: number
  readonly dataTable?: readonly (readonly string[])[]
  readonly docString?: { readonly content: string; readonly mediaType?: string }
}

export type GherkinScenario = {
  readonly featureName: string
  readonly name: string
  /** The line of the scenario, or of the example row for an outline. */
  readonly line: number
  readonly steps: readonly GherkinStep[]
}

export type Feature = {
  readonly name: string
  readonly pickles: readonly Pickle[]
  /** The line of every node of the document by its id: scenarios, example rows, steps. */
  readonly lines: ReadonlyMap<string, number>
  readonly steps: ReadonlyMap<string, Step>
}

export const parseFeature = (source: string, uri: string): Feature => {
  const newId = IdGenerator.incrementing()
  const parser = new Parser(new AstBuilder(newId), new GherkinClassicTokenMatcher())
  const document = parser.parse(source)
  const lines = new Map<string, number>()
  const steps = new Map<string, Step>()
  for (const child of document.feature?.children ?? []) indexChild(child, lines, steps)
  return {
    name: document.feature?.name ?? '',
    pickles: compile(document, uri, newId),
    lines,
    steps,
  }
}

export const readFeature = (featurePath: string): Feature =>
  parseFeature(fs.readFileSync(featurePath, 'utf8'), featurePath)

const indexChild = (
  child: FeatureChild | RuleChild,
  lines: Map<string, number>,
  steps: Map<string, Step>,
): void => {
  const node = child.background ?? child.scenario
  for (const step of node?.steps ?? []) steps.set(step.id, step)
  if (child.scenario) {
    lines.set(child.scenario.id, child.scenario.location.line)
    const rows = child.scenario.examples.flatMap((examples) => examples.tableBody)
    for (const row of rows) lines.set(row.id, row.location.line)
  }
  if ('rule' in child && child.rule) {
    for (const ruleChild of child.rule.children) indexChild(ruleChild, lines, steps)
  }
}

/** The scenario whose pickle starts at `line`: the scenario's own line, or its example row's. */
export const findScenario = (feature: Feature, line: number): GherkinScenario | undefined => {
  const pickle = feature.pickles.find((one) => getPickleLine(feature, one) === line)
  if (!pickle) return undefined
  return {
    featureName: feature.name,
    name: pickle.name,
    line,
    steps: pickle.steps.map((step) => toGherkinStep(feature, step)),
  }
}

const getPickleLine = (feature: Feature, pickle: Pickle): number | undefined =>
  feature.lines.get(pickle.astNodeIds.at(-1) ?? '')

/** The keyword and line come from the written step; the text and argument from the pickle, with an outline's values in. */
const toGherkinStep = (feature: Feature, step: PickleStep): GherkinStep => {
  const written = feature.steps.get(step.astNodeIds[0] ?? '')
  return {
    keyword: written?.keyword.trim() ?? '',
    text: step.text,
    line: written?.location.line ?? 0,
    ...readArgument(step.argument),
  }
}

const readArgument = (
  argument: PickleStepArgument | undefined,
): Pick<GherkinStep, 'dataTable' | 'docString'> => {
  if (argument?.dataTable) {
    return { dataTable: argument.dataTable.rows.map((row) => row.cells.map((cell) => cell.value)) }
  }
  if (argument?.docString) {
    const { content, mediaType } = argument.docString
    return { docString: mediaType ? { content, mediaType } : { content } }
  }
  return {}
}
