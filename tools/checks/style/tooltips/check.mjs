// Rejects a native `title` attribute on an HTML element of a Vue template. The browser draws it as
// its own tooltip; every tooltip of the app is drawn by the Tooltip component of @numen/ui.
//
//   node tools/checks/style/tooltips/check.mjs            every tracked .vue file
//   node tools/checks/style/tooltips/check.mjs a.vue      the files named

import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import { pathToFileURL } from 'node:url'
import { parse as parseVue } from '@vue/compiler-sfc'

/** Whether a template attribute node sets `title`: `title="..."`, `:title="..."` or `v-bind:title="..."`, with any modifier. */
const isTitleAttribute = (prop) =>
  (prop.type === 6 && prop.name === 'title') ||
  (prop.type === 7 && prop.name === 'bind' && prop.arg?.content === 'title')

/** The 1-based lines of the native elements in a Vue file that carry a title attribute. */
export function findNativeTitles(text) {
  const { descriptor } = parseVue(text)
  if (!descriptor.template?.ast) return []
  const lines = []
  const visit = (node) => {
    if (node.type === 1 && node.tagType === 0 && node.props.some(isTitleAttribute)) lines.push(node.loc.start.line)
    for (const child of node.children ?? []) visit(child)
  }
  visit(descriptor.template.ast)
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
