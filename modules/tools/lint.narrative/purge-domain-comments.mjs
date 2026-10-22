#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')
const pathArg = args.find(a => !a.startsWith('--'))
let targetDir = pathArg ? path.resolve(pathArg) : path.resolve(import.meta.dirname, '../../..')

const IGNORED_DIRS = new Set([
  'node_modules',
  '.git',
  'dist',
  '.direnv',
  '.claude',
  'test-results',
  'coverage',
  'testdata',
])

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue', '.go', '.proto', '.css'])

const DOMAIN_TERMS = [
  'refusal', 'refused', 'refusalReason', 'readings', 'spoken', 'spokenBy', 'speaks',
  'reflow', 'reflows', 'reflowed', 'reflowing'
]

const RE_DOMAIN_IN_COMMENT = new RegExp(`\\b(?:${DOMAIN_TERMS.join('|')})\\b`, 'i')

let totalCleanedLines = 0
let totalModifiedFiles = 0

function cleanFile(filePath) {
  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  const lines = content.split('\n')
  const newLines = []
  let modified = false
  let fileCleanedCount = 0

  let inBlockComment = false
  let blockCommentLines = []
  let blockHasViolation = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()

    // Check if we are inside a multi-line /* ... */ block comment
    if (inBlockComment) {
      blockCommentLines.push(line)
      if (RE_DOMAIN_IN_COMMENT.test(line)) {
        blockHasViolation = true
      }
      if (trimmed.includes('*/')) {
        inBlockComment = false
        if (blockHasViolation) {
          modified = true
          fileCleanedCount += blockCommentLines.length
          blockCommentLines = []
          blockHasViolation = false
        } else {
          newLines.push(...blockCommentLines)
          blockCommentLines = []
        }
      }
      continue
    }

    // Single-line // comment that contains domain violations
    if (trimmed.startsWith('//')) {
      if (RE_DOMAIN_IN_COMMENT.test(trimmed)) {
        modified = true
        fileCleanedCount++
        continue
      }
      newLines.push(line)
      continue
    }

    // HTML/Vue single-line comment <!-- ... -->
    if (trimmed.startsWith('<!--') && trimmed.endsWith('-->')) {
      if (RE_DOMAIN_IN_COMMENT.test(trimmed)) {
        modified = true
        fileCleanedCount++
        continue
      }
      newLines.push(line)
      continue
    }

    // Single-line /* ... */ or /** ... */ comment
    if (trimmed.startsWith('/*') && trimmed.includes('*/')) {
      if (RE_DOMAIN_IN_COMMENT.test(trimmed)) {
        modified = true
        fileCleanedCount++
        continue
      }
      newLines.push(line)
      continue
    }

    // Start of a multi-line /* or /** comment
    if (trimmed.startsWith('/*')) {
      inBlockComment = true
      blockCommentLines = [line]
      blockHasViolation = RE_DOMAIN_IN_COMMENT.test(line)
      continue
    }

    newLines.push(line)
  }

  if (inBlockComment && !blockHasViolation) {
    newLines.push(...blockCommentLines)
  }

  // Clean empty docblocks
  let newContent = newLines.join('\n')
  newContent = newContent.replace(/\/\*\*?\s*\*\/\n?/g, '')
  newContent = newContent.replace(/<!--\s*-->\n?/g, '')

  if (modified || newContent !== content) {
    totalModifiedFiles++
    totalCleanedLines += fileCleanedCount
    if (!isDryRun) {
      fs.writeFileSync(filePath, newContent, 'utf8')
    }
  }
}

function walk(dir) {
  let entries = []
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true })
  } catch {
    return
  }

  for (const entry of entries) {
    if (IGNORED_DIRS.has(entry.name)) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full)
    else if (entry.isFile() && EXTENSIONS.has(path.extname(entry.name))) {
      cleanFile(full)
    }
  }
}

console.log(`🧹 Running Domain Comment Purger on: ${targetDir} (dry-run: ${isDryRun})...`)
walk(targetDir)
console.log(`✨ Domain Comment Purger Completed:`)
console.log(`   - Modified files     : ${totalModifiedFiles}`)
console.log(`   - Cleaned lines      : ${totalCleanedLines}`)
