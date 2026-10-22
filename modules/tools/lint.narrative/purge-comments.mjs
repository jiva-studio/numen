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

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue', '.go', '.proto'])

const ANGLISH_IDIOMS = [
  'hands back', 'comes to', 'puts up', 'beside what', 'stands on', 'stands under',
  'where it sits', 'where it stands', 'while nothing else does', 'while it stands',
  'while being said', 'what was said', 'thread of talk', 'talk the window',
  'let go of', 'names nothing', 'what an answer comes to', 'the person has open',
  'what the person sees', 'as it stands', 'rather than ', 'instead of ',
  'which is why this exists',
  'as the file stands', 'as the file now stands', 'file now stands', 'file as it stands',
  'file stands', 'stands behind', 'stands in', 'stands over', 'stands at', 'stands between',
  'sits on', 'sits down', 'sits inside', 'sits in', 'ground a field sits',
  'nobody named', 'nobody has answered', 'person may say', 'person is told',
  'the host is told', 'window is told', 'leaves behind', 'left behind',
  'held apart', 'held open', 'held for ever', 'held to',
  'refused by the use', 'refused a call', 'refused under',
  'the words the core holds', 'code sits on crust',
  'stops offering', 'offering anything', 'item offering', 'as they were offered', 'order it was offered',
  'nothing said in it', 'with nothing said', 'found at the place given',
  'model the window lists', 'setting the window lists', 'vault answers with',
  'answers whichever vault', 'window is showing',
  'follows the hand', 'hand travels', 'hand put down', 'hand lifts', 'hand that took',
  'run of the book', 'where the run begins', 'each run stands', 'swipe turns',
  'whoever counts a page', 'owes the person an answer', 'roads that ask',
  'every road to a file', 'making a file from nothing', 'words along the line',
  'stylesheet the window wears', 'hangs the headings', 'brackets answer one question',
  'stencil a card is cut by', 'quieter than', 'younger than', 'older than the',
  'wider than', 'taller than', 'stands taller', 'larger than the room', 'larger than the material',
  'reached further than',
  'is an example of a link and not a link',
  'where a link is an example of one',
  'address a position in the text stands in',
  'read by the parser every link is read by',
  'a drawn link is followed with',
  'where the position remains in no link'
]

const RE_ANGLISH_LINE = new RegExp(`(?:${ANGLISH_IDIOMS.map(s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`, 'i')

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

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()

    // Single-line // comment that contains Anglish idioms
    if (trimmed.startsWith('//')) {
      if (RE_ANGLISH_LINE.test(trimmed)) {
        modified = true
        fileCleanedCount++
        continue
      }
    }

    // Docblock line * ... that contains Anglish idioms
    if (trimmed.startsWith('*') && !trimmed.startsWith('*/') && !trimmed.startsWith('/**')) {
      if (RE_ANGLISH_LINE.test(trimmed)) {
        modified = true
        fileCleanedCount++
        continue
      }
    }

    newLines.push(line)
  }

  // Clean empty docblock blocks /** */
  let newContent = newLines.join('\n')
  newContent = newContent.replace(/\/\*\*\s*\*\/\n?/g, '')

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

console.log(`🧹 Running Automated Comment Purger on: ${targetDir} (dry-run: ${isDryRun})...`)
if (fs.existsSync(targetDir) && fs.statSync(targetDir).isFile()) {
  cleanFile(targetDir)
} else {
  walk(targetDir)
}

console.log(`✨ Comment Purger Completed:`)
console.log(`   - Modified files     : ${totalModifiedFiles}`)
console.log(`   - Cleaned comments   : ${totalCleanedLines}`)
