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

const TITLE_REPLACEMENTS = new Map([
  ['every road to a file', 'resolves file paths'],
  ['a name the vault answers with', 'vault name responses'],
  ['a passage the vault answers with', 'vault passage responses'],
  ['a neighbourhood the vault answers with', 'vault neighbourhood responses'],
  ['is requested for a hundred names, and then the person is told', 'requests a hundred names and reports status'],
  ['is nothing where the vault answers with what is not JSON, and is returned', 'returns empty when response is not valid JSON'],
  ['leave out the rename where the window is showing no vault', 'omits rename command when no vault is active'],
  ['marks the vault the window is showing, and will not take it', 'marks active vault and ignores duplicate selection'],
  ['carries the vault that was picked, not the one the window is showing', 'uses selected vault instead of active vault'],
  ['carries the name to the vault the window is showing', 'passes name to active vault'],
  ['is not reached by the name of the vault the window is showing', 'ignores active vault name'],
  ['are the settings alone where the window is showing no vault', 'shows only settings when no vault is active'],
  ['marks the one the window is showing, and draws it all the same', 'marks active item and renders normally'],
  ['sits down to the deck standing at it', 'opens review for target deck'],
  ['sits down to nothing where that deck owes nothing', 'does not start review when deck has no due cards'],
  ['sits down to nothing where the whole vault owes nothing', 'does not start review when vault has no due cards'],
  ['sits down to the whole vault where it owes something', 'starts review for vault with due cards'],
  ['sits down to every deck of a preset, and says what it could not act on', 'reviews all preset decks and reports errors'],
  ['sits down to a vault that owes only new cards, and keeps the key from the page', 'opens vault with only new cards and consumes key event'],
  ['sits down to the whole vault on enter', 'opens review for all decks on enter'],
  ['sits down to the deck a letter remains at', 'opens review for matched deck key'],
  ['is not turned over by a hand that took it across', 'does not flip card on swipe gesture across threshold'],
  ['is discarded when it is older than the newest read issued', 'discards response when superseded by newer read request'],
  ['is nothing where the position remains in no link', 'returns null when position is not within a link'],
  ['is nothing inside code, where a link is an example of one', 'returns null for links inside code blocks'],
  ['leaves days out where the grid is drawn wider than it was requested for', 'omits extra days when rendered width exceeds requested span'],
  ['reaches back from four weeks ahead once the history is older than the grid', 'offsets history range when data exceeds grid capacity'],
  ['remains clear of the edge it is against where it is larger than the room', 'clamps tooltip position when larger than viewport bounds'],
  ['draws nothing while the work is younger than the wait', 'suppresses display until debounce delay expires'],
  ['draws nothing for an item offering nothing', 'renders nothing when item has no actions'],
  ['closes when the item stops offering actions', 'closes when item has no remaining actions'],
  ['an item offering more than two actions', 'item with multiple actions'],
  ['goes when the item it is about stops offering anything', 'dismisses when target item has no actions'],
  ['what a person is told', 'player status notifications'],
  ['gives a box wider than the window a row of one', 'assigns single row when box exceeds window width'],
  ['gives a focus wider than the window a row of one', 'assigns single row when focus exceeds window width'],
  ['grows no wider than the window, the margin kept clear', 'constrains width to window minus margins'],
  ['stays as it was placed when the window is no wider than it is', 'preserves layout when window width is constrained'],
  ['does not begin on a target that sits in no drawing', 'ignores target outside canvas bounds'],
  ['starts a gesture and not choosing the node it sits on', 'initiates drag gesture without selecting node'],
  ['sits on the trailing edge, halfway down', 'positions at trailing edge vertical center'],
  ['draws a page just left behind, in case the hand arrives back', 'pre-renders previous page for reverse navigation'],
  ['is never wider than a page is drawn', 'constrains width to page boundary'],
  ['the folder a path sits in', 'parent directory resolution'],
  ['reads the folder the path it names sits in', 'reads parent directory of target path'],
  ['say every name left behind, parted by commas', 'formats remaining names as comma-separated list'],
  ['lets the window ask each kind whether it owes the person an answer and to write out', 'queries kind handlers for pending changes before flush'],
  ['is waited on again where the goal is moved to one nobody has answered', 'reschedules review when target goal has no reviews'],
  ['is the nearest place, whether the value sits on one or between two', 'snaps to nearest point'],
  ['calls a card nobody has answered new', 'classifies unreviewed card as new'],
  ['is closed by the stencil itself, and the window is told to leave it be', 'closes stencil and suppresses window prompt'],
  ['what the window is told about a plex tab', 'plex tab notifications'],
])

let totalCleaned = 0
let totalFiles = 0

function cleanFile(filePath) {
  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  let modified = false
  let count = 0

  for (const [narrativeTitle, replacement] of TITLE_REPLACEMENTS) {
    if (content.includes(`'${narrativeTitle}'`)) {
      content = content.replaceAll(`'${narrativeTitle}'`, `'${replacement}'`)
      modified = true
      count++
    }
    if (content.includes(`"${narrativeTitle}"`)) {
      content = content.replaceAll(`"${narrativeTitle}"`, `"${replacement}"`)
      modified = true
      count++
    }
    if (content.includes(`\`${narrativeTitle}\``)) {
      content = content.replaceAll(`\`${narrativeTitle}\``, `\`${replacement}\``)
      modified = true
      count++
    }
  }

  if (modified) {
    totalFiles++
    totalCleaned += count
    if (!isDryRun) {
      fs.writeFileSync(filePath, content, 'utf8')
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
    else if (entry.isFile() && (entry.name.endsWith('.test.ts') || entry.name.endsWith('.spec.ts'))) {
      cleanFile(full)
    }
  }
}

console.log(`🧹 Running Title Purger on: ${targetDir} (dry-run: ${isDryRun})...`)
walk(targetDir)
console.log(`✨ Title Purger Completed:`)
console.log(`   - Modified files  : ${totalFiles}`)
console.log(`   - Cleaned titles  : ${totalCleaned}`)
