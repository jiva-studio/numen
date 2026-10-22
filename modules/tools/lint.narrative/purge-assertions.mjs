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

const RE_GO_TEST_ASSERTION_NARRATIVE = /\bt\.(?:Errorf|Fatalf|Fatal|Error)\s*\(\s*(["`'])([^"`']*(?:stands\s+(?:at|in|under|on|over|against|behind)|told\s+(?:to|of|apart)|never\s+told|was\s+told|held\s+by|held\s+open|held\s+apart|held\s+to|left\s+behind|nobody\s+asked|nobody\s+named|nobody\s+has\s+answered)[^"`']*)\1/gi

const EXACT_REPLACEMENTS = [
  ['the panel was told nothing about why it has no agent', 'expected error reason for missing agent'],
  ['nothing answers where the agents are told to look', 'no service responding at agent endpoint'],
  ['the announcement was left behind', 'expected announcement to be removed'],
  ['a waiter was told the vault had not settled', 'unexpected unsettled notification'],
  ['the page was told nothing about why it has no agent', 'expected error reason for missing agent'],
  ['the window was told nothing', 'expected window notification'],
  ['no text is held by both vaults, so no vector here is one the other vault still holds', 'vaults do not share vectors for distinct text'],
  ['the first half of a failed migration was left behind', 'failed migration partially applied'],
  ['schema_migrations_next was left behind by the rollback', 'schema_migrations_next table retained after rollback'],
  ['a folder of one vault was told the type of another vault\'s note at the same path', 'folder received cross-vault note type'],
  ['the fingerprint is not the one the deck stands at', 'fingerprint mismatch for deck'],
  ['the card was not among what stands in the vault', 'card not found in vault'],
  ['the page was not told to ask again', 'did not receive refresh signal'],
  ['something moved and the page was never told', 'missing update notification for moved item'],
  ['a schedule nobody has answered says it is not new', 'unanswered schedule marked as not new'],
  ['a card face nobody has answered is learned', 'unanswered card face marked as learned'],
]

const SUBSTRING_REPLACEMENTS = [
  // Patterns with placeholders
  [/the page was told %q/g, 'got error message %q'],
  [/the page was told %v/g, 'got error message %v'],
  [/the window was told %v \(%v\)/g, 'got window update %v (%v)'],
  [/the window was told \+v, want \+v/g, 'got %+v, want %+v'],
  [/the window was told %d times, wanted two/g, 'got %d notifications, want 2'],
  [/the window was told %d times, wanted once/g, 'got %d notifications, want 1'],
  [/the index was told to forget %v/g, 'forgot %v'],
  [/the index was told about %q/g, 'indexed %q'],
  [/the source still stands on a reading: \+v/g, 'source still associated with recognition: %+v'],
  [/the file is at its new path and the client was told %v/g, 'file moved to new path but client received error: %v'],
  [/chunk %d is held %d times and stands in %d rows of the coarse index/g, 'chunk %d has count %d in %d coarse index rows'],
  [/chunk %d is held %d times and stands in %d rows of vectors/g, 'chunk %d has count %d in %d vector rows'],
  [/%v cut again as %v stands on %v, want %v/g, 'previous %v cut again as %v: got %v, want %v'],
  [/the note is called %q, want the note the heading stands in/g, 'got note %q, want heading parent note'],
  [/the heading stands on line %d, want 0/g, 'got heading line %d, want 0'],
  [/it stands at %q/g, 'got path %q'],
  [/the recording stands at %q/g, 'got recording path %q'],
  [/said %q, which repeats what stands in the file/g, 'got error %q repeating file content'],
  [/%q was saved and %q stands in the file/g, 'saved %q, found %q in file'],
  [/the model was told back \+v/g, 'sent back to model: %+v'],
  [/the model was told \+v/g, 'sent to model: %+v'],
  [/the call was told back with %q/g, 'call returned with %q'],
  [/the blank stands at %d/g, 'blank token is at %d'],
  [/a run says it stands at %q/g, 'got run position %q'],
  [/the element stands at %d, want 1/g, 'got element offset %d, want 1'],
  [/the %s stands at %d and what it holds at %d/g, '%s is at %d and content at %d'],
  [/page %q stands at %d in %d bytes/g, 'page %q is at %d in %d bytes'],
  [/the page stands at %d, want %d/g, 'got page offset %d, want %d'],
  [/part %q stands at %d, outside a text of %d bytes/g, 'part %q is at offset %d, outside text length %d'],
  [/%q stands at depth %d/g, 'part %q is at depth %d'],
  [/page %d was lit and nobody asked about it/g, 'unexpected highlight on page %d'],
  [/the run stands at %d-%d/g, 'run span is %d-%d'],
  [/the fingerprint handed back stands at %d bytes, and the deck is %d/g, 'returned fingerprint has size %d, want %d'],
  [/want what the stencil stands at/g, 'want current stencil state'],
  [/want what the file stands at/g, 'want current file state'],
  [/the problem stands against card %d, which is %q/g, 'problem targets card %d (%q)'],
  [/the problem for card %d stands against \+v/g, 'problem for card %d targets %+v'],
  [/the problem stands at card %v, face %v/g, 'problem targets card %v, face %v'],
  [/the preset keeps 20 minutes a day and stands at %v/g, 'preset retention limit is %v'],
  [/the preset just made stands against %v/g, 'new preset validation errors: %v'],
  [/the deck already stands under \+v/g, 'deck already belongs to preset %+v'],
  [/the deck stands under \+v/g, 'deck belongs to preset %+v'],
  [/the deck it stands in comes to \+v/g, 'parent deck is %+v'],
  [/a card nobody reached stands at %d, in front of one that was/g, 'unreached card at %d precedes reached card'],
  [/it came to \+v, want a preset nothing stands under/g, 'got %+v, want preset with no assigned decks'],
  [/%q came to \+v, want a day nothing stands on/g, '%q got %+v, want day with no reviews'],
  [/lead = %q, want what stands under the heading/g, 'lead = %q, want content under heading'],
  [/the card stands under section %d/g, 'card is under section %d'],
  [/the answer was taken back and the card stands at \+v/g, 'answer reverted, card state is %+v'],
  [/stands at %d\.\.%d/g, 'is at %d..%d'],
  [/stands at %d learned of %d, want %d of %d/g, 'has %d learned of %d, want %d of %d'],
  [/stands at %d learned before anything was answered/g, 'has %d learned before review'],
  [/stands at %d unbegun of %d, want all of them/g, 'has %d unbegun of %d, want all'],
  [/stands at %d unbegun, want %d/g, 'has %d unbegun, want %d'],
  [/stands at %d faces, want 2/g, 'has %d faces, want 2'],
  [/stands at %d learned, want %d/g, 'has %d learned, want %d'],
  [/a card face nobody has answered counts learned under \+v/g, 'unanswered card face counts learned under %+v'],
  [/the page was not told to ask again: %v/g, 'did not receive refresh signal: %v'],
  [/a %q of %s says %q and stands over %q/g, 'a %q of %s says %q and spans %q'],

  // General fallbacks
  [/stands at\b/g, 'is at'],
  [/stands on\b/g, 'is on'],
  [/stands in\b/g, 'is in'],
  [/stands under\b/g, 'is under'],
  [/stands over\b/g, 'is over'],
  [/stands against\b/g, 'targets'],
  [/stands between\b/g, 'is between'],
  [/was left behind\b/g, 'was retained'],
  [/left behind\b/g, 'retained'],
  [/was told\b/g, 'received'],
  [/never told\b/g, 'never notified'],
  [/held to\b/g, 'bound to'],
  [/held by\b/g, 'retained by'],
  [/held open\b/g, 'kept open'],
  [/held apart\b/g, 'isolated'],
  [/told to\b/g, 'instructed to'],
  [/told of\b/g, 'notified of'],
  [/told apart\b/g, 'distinguished'],
  [/nobody asked\b/g, 'unrequested'],
  [/nobody named\b/g, 'unnamed'],
  [/nobody has answered\b/g, 'unanswered'],
]

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

  const newContent = content.replace(RE_GO_TEST_ASSERTION_NARRATIVE, (fullMatch, quote, message) => {
    let newMessage = message

    for (const [exact, repl] of EXACT_REPLACEMENTS) {
      if (newMessage === exact) {
        newMessage = repl
        break
      }
    }

    if (newMessage === message) {
      for (const [re, repl] of SUBSTRING_REPLACEMENTS) {
        newMessage = newMessage.replace(re, repl)
      }
    }

    if (newMessage !== message) {
      modified = true
      count++
      return fullMatch.replace(message, newMessage)
    }
    return fullMatch
  })

  if (modified) {
    totalFiles++
    totalCleaned += count
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
    else if (entry.isFile() && entry.name.endsWith('_test.go')) {
      cleanFile(full)
    }
  }
}

console.log(`🧹 Running Assertion Purger on: ${targetDir} (dry-run: ${isDryRun})...`)
walk(targetDir)
console.log(`✨ Assertion Purger Completed:`)
console.log(`   - Modified files     : ${totalFiles}`)
console.log(`   - Cleaned assertions : ${totalCleaned}`)
