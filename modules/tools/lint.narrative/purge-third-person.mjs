#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')

// 1. Transcription package in libs/core
const transcriptionDir = path.resolve('modules/libs/core/usecase/transcription')
if (fs.existsSync(transcriptionDir)) {
  const files = fs.readdirSync(transcriptionDir)
  for (const f of files) {
    if (f.endsWith('.go')) {
      const p = path.join(transcriptionDir, f)
      let content = fs.readFileSync(p, 'utf8')
      if (content.includes('says(')) {
        content = content.replaceAll('says(', 'correctionsMap(')
        if (!isDryRun) fs.writeFileSync(p, content, 'utf8')
        console.log(`Renamed says -> correctionsMap in ${f}`)
      }
    }
  }
}

// 2. Proofreading plugin in plugins
const proofreadingFile = path.resolve('modules/plugins/tools.proofreading/backend/service/text_test.go')
if (fs.existsSync(proofreadingFile)) {
  let content = fs.readFileSync(proofreadingFile, 'utf8')
  if (content.includes('says(')) {
    content = content.replaceAll('says(', 'newReplyModel(')
    if (!isDryRun) fs.writeFileSync(proofreadingFile, content, 'utf8')
    console.log(`Renamed says -> newReplyModel in text_test.go`)
  }
}

console.log('✨ Third-person functions purge complete!')
