#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')

const replacements = [
  {
    file: 'modules/libs/core/usecase/transcription/transcribe_test.go',
    from: 'nothing says what listened',
    to: 'transcriber returned error',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/flashcards/deck_budget_test.go',
    from: 'on the front door and hands over',
    to: 'on the overview and yields',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/service/review_cards_due_test.go',
    from: 'the front door said nothing',
    to: 'overview stream empty',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/service/review_cards_due_test.go',
    from: 'the front door opens on %d of %d vaults',
    to: 'overview returned %d of %d vaults',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/service/review_keepalive_test.go',
    from: 'the front door never opened',
    to: 'overview stream never opened',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/service/review_keepalive_test.go',
    from: 'the front door opened on %+v',
    to: 'overview stream received %+v',
  },
  {
    file: 'modules/plugins/views.flashcards.studio/backend/service/review_streams_test.go',
    from: 'the front door never opened',
    to: 'overview stream never opened',
  },
  {
    file: 'modules/apps/desktop/flashcards/src/pages/session/model/session.test.ts',
    from: 'measures how long the card stood there by the clock on the wall',
    to: 'measures card review duration using timer',
  },
  {
    file: 'modules/apps/desktop/flashcards/src/pages/session/ui/Session.vue',
    from: 'a person answering with the',
    to: 'user review input with the',
  },
  {
    file: 'modules/apps/desktop/flashcards/src/pages/vaults/Vaults.vue.test.ts',
    from: "describe('the front door before it knows which vaults there are'",
    to: "describe('vault overview before loading vaults'",
  },
  {
    file: 'modules/apps/desktop/flashcards/src/pages/vaults/Vaults.vue.test.ts',
    from: "describe('the front door while the vaults are being counted'",
    to: "describe('vault overview while loading counts'",
  },
  {
    file: 'modules/apps/desktop/flashcards/src/pages/vaults/Vaults.vue.test.ts',
    from: "describe('the front door once the counts are in'",
    to: "describe('vault overview once counts are loaded'",
  },
  {
    file: 'modules/apps/docs/src/content/docs/review.mdx',
    from: 'the row on the front door and the session',
    to: 'the row on the overview and the session',
  },
]

let count = 0
for (const { file, from, to } of replacements) {
  const p = path.resolve(file)
  if (fs.existsSync(p)) {
    let content = fs.readFileSync(p, 'utf8')
    if (content.includes(from)) {
      content = content.replaceAll(from, to)
      if (!isDryRun) fs.writeFileSync(p, content, 'utf8')
      console.log(`Replaced in ${file}: "${from}" -> "${to}"`)
      count++
    }
  }
}

console.log(`✨ Replaced ${count} domestic metaphor occurrences.`)
