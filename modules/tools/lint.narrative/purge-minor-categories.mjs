#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')

// 1. Error storytelling in Go
const errorReplacements = [
  {
    file: 'modules/libs/core/adapter/mcp/view_test.go',
    from: 'errors.New("nobody is looking")',
    to: 'errors.New("no active view")',
  },
  {
    file: 'modules/libs/core/usecase/note/focus_note_test.go',
    from: 'errors.New("nobody is looking")',
    to: 'errors.New("no active view")',
  },
  {
    file: 'modules/libs/plugin-sdk-go/tools/tools_test.go',
    from: 'errors.New("nobody to greet")',
    to: 'errors.New("recipient required")',
  },
  {
    file: 'modules/libs/plugin-sdk-go/tools/tools_test.go',
    from: 'answer.GetResult().GetText() != "nobody to greet"',
    to: 'answer.GetResult().GetText() != "recipient required"',
  },
]

for (const { file, from, to } of errorReplacements) {
  const p = path.resolve(file)
  if (fs.existsSync(p)) {
    let content = fs.readFileSync(p, 'utf8')
    if (content.includes(from)) {
      content = content.replace(from, to)
      if (!isDryRun) fs.writeFileSync(p, content, 'utf8')
      console.log(`Replaced error storytelling in ${file}`)
    }
  }
}

// 2. Gerund variable in windowTabs.test.ts
const windowTabsTest = path.resolve('modules/apps/desktop/editor/src/entities/tab/model/windowTabs.test.ts')
if (fs.existsSync(windowTabsTest)) {
  let content = fs.readFileSync(windowTabsTest, 'utf8')
  if (content.includes('const offering =')) {
    content = content.replaceAll('const offering =', 'const offerTab =')
    content = content.replaceAll('offering(', 'offerTab(')
    if (!isDryRun) fs.writeFileSync(windowTabsTest, content, 'utf8')
    console.log(`Renamed gerund variable offering -> offerTab in windowTabs.test.ts`)
  }
}

console.log('✨ Minor categories purge complete!')
