/**
 * The Tailwind utilities a set of frontend files names, built with the theme of `@numen/ui`. The build of
 * a plugin and the stories of every plugin read them alike.
 */
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, extname } from 'node:path'
import { compile } from '@tailwindcss/node'
import { Scanner } from '@tailwindcss/oxide'
import { parse as parseSfc } from '@vue/compiler-sfc'
import { getNormalCss } from '@numen/ui/vite'

/** The stylesheet of `@numen/ui` the utilities are built from. */
const THEME = createRequire(import.meta.url).resolve('@numen/ui/theme.css')

function getVueScanContent(source: string): string {
  const { descriptor } = parseSfc(source)
  return [descriptor.template?.content, descriptor.script?.content, descriptor.scriptSetup?.content]
    .filter(Boolean)
    .join('\n')
}

/** The class names Tailwind finds in the files; of a Vue file only the template and the scripts are read. */
export function getCandidates(files: readonly string[]): string[] {
  const scanner = new Scanner({})
  return scanner.scanFiles(
    files.map((file) => {
      const content = readFileSync(file, 'utf8')
      const extension = extname(file).slice(1)
      return {
        content: extension === 'vue' ? getVueScanContent(content) : content,
        extension,
      }
    }),
  )
}

/** The CSS of the utilities among the candidates, with the theme of `@numen/ui`. */
export async function getUtilityCss(candidates: readonly string[]): Promise<string> {
  const compiler = await compile(`@import ${JSON.stringify(THEME)};`, {
    base: dirname(THEME),
    onDependency: () => undefined,
  })
  return getNormalCss(compiler.build([...candidates]))
}
