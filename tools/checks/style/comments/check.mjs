// Rejects comments that repeat what a name and a type already state: a comment on a field or a
// property, and a doc comment on a function that is not exported. Go files go to the Go command
// beside this script; TypeScript and Vue files are read here with the TypeScript parser.
//
//   node tools/checks/style/comments/check.mjs            every tracked file
//   node tools/checks/style/comments/check.mjs --staged   the files staged for the next commit
//   node tools/checks/style/comments/check.mjs a.ts b.go  the files named
//   add --fix to remove the reported comments instead of failing
//
// Directives (eslint, @ts-, Stryker, istanbul, prettier), section banners and @deprecated are
// never reported. Go types that are documentation for someone else carry a marker line instead
// (see main.go).

import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const HERE = path.dirname(fileURLToPath(import.meta.url))
const SCRIPT_FILE = /\.(ts|tsx|mts|cts|vue)$/
/** The most lines of text a doc comment on an exported symbol may have. */
export const MAX_DOC_LINES = 5
const KEEP =
  /^(\/\/\/?|\/\*\*?)\s*(eslint|@ts-|Stryker|istanbul|prettier|c8 |v8 |@vite|webpack|#region|#endregion|<reference|global )|@deprecated|@vitest-environment|^(\/\/|\/\*)\s*[-=─*]{6,}/
const GENERATED = /^[ \t]*(\/\/|#|\/\*)\s*(Code generated .* DO NOT EDIT|@generated)/m

function isGenerated(text) {
  return GENERATED.test(text.split('\n').slice(0, 20).join('\n'))
}

/** Returns the lines of text of a JSDoc block: the lines between the markers, blank ones included. */
function countDocLines(block) {
  const lines = block
    .replace(/^\/\*\*/, '')
    .replace(/\*\/$/, '')
    .split('\n')
    .map((line) => line.replace(/^\s*\*? ?/, '').trim())
  if (lines[0] === '') lines.shift()
  if (lines.length > 0 && lines.at(-1) === '') lines.pop()
  return lines.length
}

function getScriptBlocks(file, text) {
  if (!file.endsWith('.vue')) return [{ offset: 0, content: text }]
  const { parse } = require('@vue/compiler-sfc')
  const { descriptor } = parse(text)
  return [descriptor.script, descriptor.scriptSetup]
    .filter(Boolean)
    .map((block) => ({ offset: block.loc.start.offset, content: block.content }))
}

/**
 * Returns the rejected comments of one TypeScript or Vue source, each with its line, message and
 * character range in the source. A finding with `long` set is a doc comment over MAX_DOC_LINES
 * that `--fix` leaves in place.
 */
export function findScriptComments(file, text) {
  if (isGenerated(text)) return []
  const ts = require('typescript')
  const findings = []
  const seen = new Set()
  for (const { offset, content } of getScriptBlocks(file, text)) {
    const kind = file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS
    const source = ts.createSourceFile(file, content, ts.ScriptTarget.Latest, true, kind)
    const report = (node, message, trailing) => {
      const leading = ts.getLeadingCommentRanges(content, node.getFullStart()) ?? []
      const ranges = []
      let next = node.getStart(source)
      for (let k = leading.length - 1; k >= 0; k--) {
        const gap = content.slice(leading[k].end, next)
        if ((gap.match(/\n/g) ?? []).length > 1) break
        ranges.unshift(leading[k])
        next = leading[k].pos
      }
      if (trailing) {
        ranges.push(...(ts.getTrailingCommentRanges(content, node.end) ?? []))
        if (content[node.end] === ',' || content[node.end] === ';') {
          ranges.push(...(ts.getTrailingCommentRanges(content, node.end + 1) ?? []))
        }
      }
      for (const range of ranges) {
        if (KEEP.test(content.slice(range.pos, range.end)) || seen.has(offset + range.pos)) continue
        seen.add(offset + range.pos)
        const line = text.slice(0, offset + range.pos).split('\n').length
        findings.push({ file, line, message, from: offset + range.pos, to: offset + range.end })
      }
    }
    const reportLength = (node) => {
      let name
      if (ts.isVariableStatement(node)) {
        name = node.declarationList.declarations[0]?.name.getText(source)
      } else if (
        ts.isFunctionDeclaration(node) ||
        ts.isClassDeclaration(node) ||
        ts.isInterfaceDeclaration(node) ||
        ts.isTypeAliasDeclaration(node) ||
        ts.isEnumDeclaration(node)
      ) {
        name = node.name?.getText(source) ?? 'default'
      }
      if (name === undefined) return
      const leading = ts.getLeadingCommentRanges(content, node.getFullStart()) ?? []
      const doc = leading.at(-1)
      if (!doc || doc.kind !== ts.SyntaxKind.MultiLineCommentTrivia) return
      if ((content.slice(doc.end, node.getStart(source)).match(/\n/g) ?? []).length > 1) return
      const raw = content.slice(doc.pos, doc.end)
      if (!raw.startsWith('/**') || KEEP.test(raw)) return
      const count = countDocLines(raw)
      if (count <= MAX_DOC_LINES) return
      findings.push({
        file,
        line: text.slice(0, offset + doc.pos).split('\n').length,
        message: `doc comment on exported ${name} has ${count} lines; A doc comment is at most five lines: say the fact the name and type cannot show`,
        from: offset + doc.pos,
        to: offset + doc.end,
        long: true,
      })
    }
    const isExported = (node) => (ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export) !== 0
    const nameOf = (node) => node.name?.getText(source) ?? 'member'
    const visit = (node) => {
      if (ts.isPropertySignature(node) || ts.isPropertyDeclaration(node)) {
        report(
          node,
          `comment on property ${nameOf(node)}; the name and type state what it holds`,
          true,
        )
      } else if (
        ts.isCallExpression(node) &&
        /^define(Props|Emits)$/.test(node.expression.getText(source)) &&
        node.arguments[0] &&
        ts.isObjectLiteralExpression(node.arguments[0])
      ) {
        for (const member of node.arguments[0].properties) {
          report(
            member,
            `comment on prop ${nameOf(member)}; the name and type state what it holds`,
            true,
          )
        }
      } else if (
        ts.isMethodDeclaration(node) &&
        (ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Private ||
          ts.isPrivateIdentifier(node.name))
      ) {
        report(
          node,
          `doc comment on private method ${nameOf(node)}; the name and signature state what it does`,
          false,
        )
      }
      if (node.parent === source && isExported(node)) reportLength(node)
      if (node.parent === source) {
        const isDefault = (node.modifiers ?? []).some(
          (m) => m.kind === ts.SyntaxKind.DefaultKeyword,
        )
        if (ts.isFunctionDeclaration(node) && !isExported(node) && !isDefault) {
          report(
            node,
            `doc comment on unexported function ${nameOf(node)}; the name and signature state what it does`,
            false,
          )
        } else if (
          ts.isVariableStatement(node) &&
          !isExported(node) &&
          node.declarationList.declarations.length === 1
        ) {
          const [declaration] = node.declarationList.declarations
          const init = declaration.initializer
          if (init && (ts.isArrowFunction(init) || ts.isFunctionExpression(init))) {
            report(
              node,
              `doc comment on unexported function ${nameOf(declaration)}; the name and signature state what it does`,
              false,
            )
          }
        }
      }
      ts.forEachChild(node, visit)
    }
    visit(source)
  }
  return findings
}

/** Returns the text with the given character ranges removed; a line left blank is removed whole. */
export function removeRanges(text, ranges) {
  let out = text
  for (const { from, to } of [...ranges].sort((a, b) => b.from - a.from)) {
    let start = from
    while (start > 0 && (out[start - 1] === ' ' || out[start - 1] === '\t')) start--
    let end = to
    while (end < out.length && (out[end] === ' ' || out[end] === '\t')) end++
    const atLineStart = start === 0 || out[start - 1] === '\n'
    const atLineEnd = end >= out.length || out[end] === '\n'
    if (atLineStart && atLineEnd)
      out = out.slice(0, start) + out.slice(end < out.length ? end + 1 : end)
    else if (atLineEnd) out = out.slice(0, start) + out.slice(to)
    else out = out.slice(0, from) + out.slice(end)
  }
  return out
}

function getFiles(root, args) {
  const named = args.filter((arg) => !arg.startsWith('--'))
  if (named.length > 0) return named
  const command = args.includes('--staged')
    ? ['diff', '--cached', '--name-only', '--diff-filter=ACM']
    : ['ls-files']
  const run = spawnSync('git', command, { cwd: root, encoding: 'utf8', maxBuffer: 1 << 28 })
  if (run.status !== 0) throw new Error(`git ${command.join(' ')}: ${run.stderr}`)
  return run.stdout.split('\n').filter(Boolean)
}

function checkGo(root, files, fix) {
  if (files.length === 0) return 0
  const run = spawnSync('go', ['run', '.', ...(fix ? ['-fix'] : [])], {
    cwd: HERE,
    env: { ...process.env, GOWORK: 'off' },
    input: files.map((file) => path.resolve(root, file)).join('\n'),
    encoding: 'utf8',
  })
  process.stdout.write(run.stdout.replaceAll(`${root}/`, ''))
  process.stderr.write(run.stderr.replace(/^exit status \d+\n/m, ''))
  return run.status ?? 2
}

function main() {
  const args = process.argv.slice(2)
  const fix = args.includes('--fix')
  const root = spawnSync('git', ['rev-parse', '--show-toplevel'], {
    encoding: 'utf8',
  }).stdout.trim()
  const files = getFiles(root, args).map((file) => path.relative(root, path.resolve(root, file)))
  let failed = false
  for (const file of files.filter((f) => SCRIPT_FILE.test(f) && !f.endsWith('.d.ts'))) {
    const absolute = path.join(root, file)
    if (!fs.existsSync(absolute)) continue
    const text = fs.readFileSync(absolute, 'utf8')
    const found = findScriptComments(file, text)
    if (found.length === 0) continue
    if (fix) {
      const removable = found.filter((finding) => !finding.long)
      if (removable.length > 0) fs.writeFileSync(absolute, removeRanges(text, removable))
    }
    const findings = fix ? found.filter((finding) => finding.long) : found
    if (findings.length === 0) continue
    failed = true
    for (const finding of findings)
      console.log(`${finding.file}:${finding.line}: ${finding.message}`)
  }
  const goStatus = checkGo(
    root,
    files.filter((f) => f.endsWith('.go')),
    fix,
  )
  if (goStatus > 1) process.exit(goStatus)
  if (failed || goStatus === 1) {
    console.log(
      '\nA name and a type state what a field holds and what an unexported function does.',
    )
    console.log('Put a fact they cannot show in the comment of the type or the exported function.')
    console.log('`node tools/checks/style/comments/check.mjs --fix <files>` removes the reported comments.')
    process.exit(1)
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main()
