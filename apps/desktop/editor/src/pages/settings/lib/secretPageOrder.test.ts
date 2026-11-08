/**
 * The order of the sections on the page of secrets, from the cases the core checks its own order
 * with, so the two cannot drift apart.
 */
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { createPluginInfo } from '@/testing/installation'
import { createSecretSections } from './secretSections'

interface PageOrderCase {
  readonly name: string
  readonly asked: readonly string[]
  readonly secrets: readonly { readonly id: string; readonly group: string }[]
  readonly expected: readonly string[]
}

const here = dirname(fileURLToPath(import.meta.url))
const file = join(here, '../../../../../../../libs/core/usecase/secret/testdata/page_order.json')
const cases = JSON.parse(readFileSync(file, 'utf8')) as PageOrderCase[]

describe('the order of the page of secrets', () => {
  it.each(cases)('is the order of the core: $name', (one) => {
    const plugin = createPluginInfo('md.numen.sample', 'Sample', {
      secrets: one.asked.map((group) => ({
        name: 'apiKey',
        title: 'Key',
        description: 'A key',
        group,
      })),
    })
    const secrets = one.secrets.map((entry) => ({
      id: entry.id,
      name: entry.id,
      group: entry.group,
      allowed: [],
      hasValue: true,
    }))
    const ids = createSecretSections([plugin], secrets).flatMap((section) =>
      section.secrets.map((entry) => entry.id),
    )
    expect(ids).toStrictEqual(one.expected)
  })
})
