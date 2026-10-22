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

let totalModifiedFiles = 0
let totalReplacements = 0

function cleanFile(filePath) {
  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  if (!content.includes('reflowed')) return

  const newContent = content.replace(/\breflowed\b/g, 'sampleBook')
  if (newContent !== content) {
    totalModifiedFiles++
    const count = (content.match(/\breflowed\b/g) || []).length
    totalReplacements += count
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
    else if (entry.isFile() && (entry.name.endsWith('.go') || entry.name.endsWith('.ts'))) {
      cleanFile(full)
    }
  }
}

console.log(`🧹 Running Reflow Domain Purger on: ${targetDir} (dry-run: ${isDryRun})...`)
walk(targetDir)
console.log(`✨ Reflow Domain Purger Completed:`)
console.log(`   - Modified files       : ${totalModifiedFiles}`)
console.log(`   - Renamed 'reflowed'   : ${totalReplacements}`)
