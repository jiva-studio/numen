/**
 * The `registerCommand` calls of a plugin frontend, read from the syntax tree of each source file.
 *
 * The options of a call are an object literal written in the call, or an object literal a constant of
 * the same file holds. The id is a string literal, or a constant of the same file that holds one.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import ts from 'typescript'

/** The name a registration lists for options this cannot read property by property. */
export const UNREADABLE_FIELD = '<unreadable>'

const unwrap = (node) =>
  node && (ts.isAsExpression(node) || ts.isSatisfiesExpression(node) || ts.isParenthesizedExpression(node))
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
  if (value && (ts.isStringLiteral(value) || ts.isNoSubstitutionTemplateLiteral(value))) return value.text
  if (value && ts.isIdentifier(value)) {
    const constant = findConstant(sourceFile, value.text)
    if (constant && (ts.isStringLiteral(constant) || ts.isNoSubstitutionTemplateLiteral(constant))) {
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
  (ts.isIdentifier(callee) && callee.text === 'registerCommand') ||
  (ts.isPropertyAccessExpression(callee) && callee.name.text === 'registerCommand')

/**
 * Every `registerCommand` call of the source, as `{ fields, id }`: the names of the options the call
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

/** Every `registerCommand` call under a frontend folder, as `{ file, fields, id }`. */
export const listRegistrations = (frontendDir) =>
  listSources(frontendDir).flatMap((file) =>
    readRegistrations(readFileSync(file, 'utf8'), file).map((call) => ({ file, ...call })),
  )
