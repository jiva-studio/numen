// Rejects a native `title` attribute on an HTML element of a Vue template. The browser draws it as
// its own tooltip; every tooltip of the app is drawn by the Tooltip component of @numen/ui.
//
//   node tools/checks/style/tooltips/check.mjs            every tracked .vue file
//   node tools/checks/style/tooltips/check.mjs a.vue      the files named

import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import { pathToFileURL } from 'node:url'

const OPENING_TAG = /<([a-z][a-z0-9]*)((?:\s+(?:"[^"]*"|'[^']*'|[^>"'])*)?)>/g
const TITLE_ATTRIBUTE = /(?:^|\s)(?::|v-bind:)?title(?=[\s=/]|$)/

/** The 1-based lines of the native elements in a Vue file that carry a title attribute. */
export function findNativeTitles(text) {
  const template = text.match(/<template[\s>][\s\S]*<\/template>/)
  if (!template) return []
  const offset = template.index
  const lines = []
  for (const tag of template[0].matchAll(OPENING_TAG)) {
    if (!TITLE_ATTRIBUTE.test(tag[2])) continue
    const position = offset + tag.index
    lines.push(text.slice(0, position).split('\n').length)
  }
  return lines
}

function listFiles(args) {
  if (args.length > 0) return args
  return execFileSync('git', ['ls-files', '*.vue'], { encoding: 'utf8' }).split('\n').filter(Boolean)
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? '').href) {
  let failures = 0
  for (const file of listFiles(process.argv.slice(2))) {
    for (const line of findNativeTitles(fs.readFileSync(file, 'utf8'))) {
      console.log(`${file}:${line}: a native title attribute draws the browser's tooltip: use the Tooltip component`)
      failures++
    }
  }
  process.exit(failures > 0 ? 1 : 0)
}
