/**
 * The command registrations of a plugin frontend (`register.command`) and backend (`Registry.Add`).
 *
 * Frontend registrations are read from TypeScript syntax trees; backend registrations are read
 * with `go/ast` through `backend_commands.go`.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'
import { PLUGINS_DIR, readPluginManifests } from './manifests.mjs'
import { execTool } from './tools.mjs'

/** The name a registration lists for options this cannot read property by property. */
export const UNREADABLE_FIELD = '<unreadable>'

const SHARED_DIR = fileURLToPath(new URL('.', import.meta.url))
const COMMAND_CONSTANTS_FILE = fileURLToPath(
  new URL('../../libs/plugin/sdk-go/command/shared_commands.go', import.meta.url),
)

const unwrap = (node) =>
  node &&
  (ts.isAsExpression(node) || ts.isSatisfiesExpression(node) || ts.isParenthesizedExpression(node))
    ? unwrap(node.expression)
    : node

/** The initializer of the constant `name` declared anywhere in the file, or undefined. */
const findConstant = (sourceFile, name) => {
  let found
  const visit = (node) => {
    if (found !== undefined) return
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.name.text === name) {
      found = unwrap(node.initializer)
      return
    }
    ts.forEachChild(node, visit)
  }
  visit(sourceFile)
  return found
}

const getText = (sourceFile, node) => {
  const value = unwrap(node)
  if (value && (ts.isStringLiteral(value) || ts.isNoSubstitutionTemplateLiteral(value)))
    return value.text
  if (value && ts.isIdentifier(value)) {
    const constant = findConstant(sourceFile, value.text)
    if (
      constant &&
      (ts.isStringLiteral(constant) || ts.isNoSubstitutionTemplateLiteral(constant))
    ) {
      return constant.text
    }
  }
  return undefined
}

const getOptions = (sourceFile, argument) => {
  const value = unwrap(argument)
  if (value && ts.isObjectLiteralExpression(value)) return value
  if (value && ts.isIdentifier(value)) {
    const constant = findConstant(sourceFile, value.text)
    if (constant && ts.isObjectLiteralExpression(constant)) return constant
  }
  return undefined
}

const getPropertyName = (property) => {
  if (ts.isShorthandPropertyAssignment(property)) return property.name.text
  if (ts.isSpreadAssignment(property)) return UNREADABLE_FIELD
  const name = property.name
  return name && (ts.isIdentifier(name) || ts.isStringLiteral(name)) ? name.text : UNREADABLE_FIELD
}

const isRegisterCommand = (callee) =>
  ts.isPropertyAccessExpression(callee) &&
  callee.name.text === 'command' &&
  ((ts.isIdentifier(callee.expression) && callee.expression.text === 'register') ||
    (ts.isPropertyAccessExpression(callee.expression) && callee.expression.name.text === 'register'))

/**
 * Every `register.command` call of the source, as `{ fields, id }`: the names of the options the call
 * passes, and the id, or undefined where it is not a string this can read.
 *
 * @param {string} source - the text of a TypeScript file
 * @param {string} fileName - the name the file is parsed under
 */
export const readRegistrations = (source, fileName = 'source.ts') => {
  const sourceFile = ts.createSourceFile(fileName, source, ts.ScriptTarget.Latest, true)
  const found = []
  const visit = (node) => {
    if (ts.isCallExpression(node) && isRegisterCommand(node.expression)) {
      const options = node.arguments[0] && getOptions(sourceFile, node.arguments[0])
      const idProperty = options?.properties.find(
        (one) => ts.isPropertyAssignment(one) && getPropertyName(one) === 'id',
      )
      const shorthandId = options?.properties.find(
        (one) => ts.isShorthandPropertyAssignment(one) && one.name.text === 'id',
      )
      found.push({
        fields: options ? options.properties.map(getPropertyName) : [UNREADABLE_FIELD],
        id: idProperty
          ? getText(sourceFile, idProperty.initializer)
          : shorthandId
            ? getText(sourceFile, shorthandId.name)
            : undefined,
      })
    }
    ts.forEachChild(node, visit)
  }
  visit(sourceFile)
  return found
}

const isSource = (name) => /\.ts$/.test(name) && !/\.(test|stories)\.ts$/.test(name)

/** The TypeScript sources of a frontend folder, tests and stories left out, in path order. */
export const listSources = (dir) =>
  readdirSync(dir, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name))
    .flatMap((entry) => {
      const path = join(dir, entry.name)
      if (entry.isDirectory()) return entry.name === 'node_modules' ? [] : listSources(path)
      return isSource(entry.name) ? [path] : []
    })

/** Every `register.command` call under a frontend folder, as `{ file, fields, id }`. */
export const listRegistrations = (frontendDir) =>
  listSources(frontendDir).flatMap((file) =>
    readRegistrations(readFileSync(file, 'utf8'), file).map((call) => ({ file, ...call })),
  )

/** The Go sources of a backend folder, tests left out, in path order. */
export const listBackendSources = (dir) =>
  readdirSync(dir, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name))
    .flatMap((entry) => {
      const path = join(dir, entry.name)
      if (entry.isDirectory()) return entry.name === 'node_modules' ? [] : listBackendSources(path)
      return entry.name.endsWith('.go') && !entry.name.endsWith('_test.go') ? [path] : []
    })

/**
 * Every command id a plugin backend adds to a `command.Registry`, read from the Go syntax trees of
 * `files` as `{ file, id }`.
 */
export const readBackendRegistrations = (files) => {
  if (files.length === 0) return []
  const output = execTool('go', ['run', 'backend_commands.go'], {
    cwd: SHARED_DIR,
    env: { ...process.env, GOWORK: 'off' },
    input: [COMMAND_CONSTANTS_FILE, ...files].join('\n'),
    encoding: 'utf8',
  })
  return JSON.parse(output)
}

/** Every command registration under a plugin's `backend/` folder, as `{ file, id }`. */
export const listBackendRegistrations = (backendDir) =>
  existsSync(backendDir) ? readBackendRegistrations(listBackendSources(backendDir)) : []

/**
 * Checks that every command (`contributes.commands`) and shared command handler
 * (`contributes.handlers`) declared in `manifest.json` has a handler registered in the plugin's
 * frontend or backend, and that every handler registered in the plugin's frontend or backend is
 * declared in `manifest.json`. Returns one error line per mismatch, naming the plugin, the id and
 * the missing side.
 */
export const findPluginCommandMismatches = (pluginsDir = undefined) => {
  const manifests = readPluginManifests(pluginsDir)
  const root = pluginsDir ?? PLUGINS_DIR
  const frontendFiles = []
  const backendFiles = []
  for (const { name } of manifests) {
    const feDir = join(root, name, 'frontend')
    if (existsSync(feDir)) frontendFiles.push(...listSources(feDir))
    const beDir = join(root, name, 'backend')
    if (existsSync(beDir)) backendFiles.push(...listBackendSources(beDir))
  }
  const frontendCalls = frontendFiles.flatMap((file) =>
    readRegistrations(readFileSync(file, 'utf8'), file).map((call) => ({ file, ...call })),
  )
  const backendCalls = readBackendRegistrations(backendFiles)
  const issues = []
  for (const { name, document } of manifests) {
    const prefix = join(root, name) + '/'
    const feForPlugin = frontendCalls.filter((call) => call.file.startsWith(prefix))
    const beForPlugin = backendCalls.filter((call) => call.file.startsWith(prefix))
    const declaredCommands = (document.contributes?.commands ?? []).map((one) => one.id)
    const declaredHandlers = (document.contributes?.handlers ?? []).map((one) => one.command)
    const declared = new Set([...declaredCommands, ...declaredHandlers])
    const registered = new Set()
    for (const { file, id } of feForPlugin) {
      if (!id) {
        issues.push(`${name}: frontend (${file}) registers a command with an unreadable id`)
        continue
      }
      registered.add(id)
      if (!declared.has(id)) {
        issues.push(
          `${name}: command "${id}" is registered in frontend (${file}), missing from manifest`,
        )
      }
    }
    for (const { file, id } of beForPlugin) {
      if (!id || id === UNREADABLE_FIELD) {
        issues.push(`${name}: backend (${file}) registers a command with an unreadable id`)
        continue
      }
      registered.add(id)
      if (!declared.has(id)) {
        issues.push(
          `${name}: command "${id}" is registered in backend (${file}), missing from manifest`,
        )
      }
    }
    for (const id of declaredCommands) {
      if (!registered.has(id)) {
        issues.push(
          `${name}: command "${id}" is declared in manifest.contributes.commands, missing handler in code`,
        )
      }
    }
    for (const id of declaredHandlers) {
      if (!registered.has(id)) {
        issues.push(
          `${name}: handler "${id}" is declared in manifest.contributes.handlers, missing handler in code`,
        )
      }
    }
  }
  return issues
}
