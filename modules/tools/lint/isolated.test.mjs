import assert from 'node:assert/strict'
import test from 'node:test'
import { sources } from './source.mjs'
import { ISOLATED, imports, isRefused, refused } from './isolated.mjs'

/**
 * The rule the component module exists to keep. It held by memory alone until
 * now: nothing read the imports, so the first anyone would hear of a component
 * reaching the schema is a library that cannot be built without it.
 */
test('the component module reaches nothing of the application', () => {
  const files = sources(['.ts', '.vue'])
  assert.deepEqual(refused(files), [])

  // A walk that read none of the module is a rule checked against nothing.
  const read = files.filter((one) => one.at.startsWith(`${ISOLATED}/`))
  assert.ok(read.length > 200, `${read.length} files read: the walk is not reading the module`)
  assert.ok(
    read.some((one) => one.at.endsWith('.vue')),
    'the walk read no component, so the rule stops at the templates',
  )
})

/** What counts as reaching the application, and what does not. */
test('what the isolation rule reads', () => {
  const reached = [
    "import { Role } from '@numen/protocol'",
    "import type { Result } from '@numen/wire'",
    "import { core } from '@numen/editor'",
    "import { core } from '../../../apps/desktop/editor/src/app/vault'",
    "const held = await import('@numen/protocol')",
    "export type { Span } from '@numen/protocol'",
  ]
  for (const one of reached) {
    assert.ok(
      imports(one).some(isRefused),
      `${one} reaches the application and the rule did not see it`,
    )
  }

  const allowed = [
    "import { computed } from 'vue'",
    "import { useViewport } from '@/shared/lib/viewport'",
    "import type { Mark } from '../lib/spread'",
    "import { EditorView } from '@codemirror/view'",
  ]
  for (const one of allowed) {
    assert.ok(
      !imports(one).some(isRefused),
      `${one} reaches nothing of the application and the rule refused it`,
    )
  }
})

/** A line written about an import is not an import. */
test('a commented import is not a reach', () => {
  const said = "// import { Role } from '@numen/protocol'\nimport { computed } from 'vue'"
  assert.deepEqual(imports(said), ['vue'])
})

/** The rule is read off the script of a component and not off its markup. */
test('a component is read through its script', () => {
  const said = [
    '<script setup lang="ts">',
    "import { computed } from 'vue'",
    '</script>',
    "<template><p>from '@numen/protocol'</p></template>",
  ].join('\n')
  assert.deepEqual(imports(said), ['vue'])
})
