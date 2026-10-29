/**
 * The JSON fixtures in `testdata/api` are the messages of the contract between the backend and this
 * frontend. Each is checked against the interfaces of `types.ts` and sent through the client.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import typesSource from './types.ts?raw'

const fixtureFiles = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  query: '?raw',
  import: 'default',
}) as Record<string, string>

interface Field {
  readonly isOptional: boolean
  readonly type: string
  readonly isList: boolean
}

/** The fields of each interface and the values of each enumeration declared in `types.ts`. */
const parseTypes = (source: string) => {
  const interfaces = new Map<string, Map<string, Field>>()
  const enums = new Map<string, readonly string[]>()
  for (const match of source.matchAll(/export interface (\w+) \{([^}]*)\}/g)) {
    const fields = new Map<string, Field>()
    for (const line of (match[2] ?? '').split('\n')) {
      const field = /^\s*(\w+)(\?)?: ([\w<>, ]+?)(\[\])?\s*$/.exec(line)
      if (field) {
        fields.set(field[1] ?? '', {
          isOptional: field[2] === '?',
          type: field[3] ?? '',
          isList: field[4] === '[]',
        })
      }
    }
    interfaces.set(match[1] ?? '', fields)
  }
  for (const match of source.matchAll(/export const (\w+) = \{([^}]*)\} as const/g)) {
    enums.set(
      match[1] ?? '',
      [...(match[2] ?? '').matchAll(/: '([^']*)'/g)].map((value) => value[1] ?? ''),
    )
  }
  for (const match of source.matchAll(/export type (\w+) = Record<string, never>/g)) {
    interfaces.set(match[1] ?? '', new Map())
  }
  return { interfaces, enums }
}

const { interfaces, enums } = parseTypes(typesSource)

/** The problems of a value against the interface, one text for each. */
const check = (value: unknown, typeName: string, path: string, isRequest: boolean): string[] => {
  if (typeName === 'string') return typeof value === 'string' ? [] : [`${path}: not a string`]
  if (typeName === 'number') return typeof value === 'number' ? [] : [`${path}: not a number`]
  if (typeName === 'boolean') return typeof value === 'boolean' ? [] : [`${path}: not a boolean`]
  if (typeName.startsWith('Record<')) {
    return typeof value === 'object' && value !== null ? [] : [`${path}: not an object`]
  }
  const values = enums.get(typeName)
  if (values)
    return values.includes(value as string)
      ? []
      : [`${path}: ${String(value)} is not a ${typeName}`]
  const fields = interfaces.get(typeName)
  if (!fields) return [`${path}: unknown type ${typeName}`]
  if (typeof value !== 'object' || value === null) return [`${path}: not an object`]
  const problems: string[] = []
  const record = value as Record<string, unknown>
  for (const key of Object.keys(record)) {
    if (!fields.has(key)) problems.push(`${path}.${key}: not declared by ${typeName}`)
  }
  for (const [key, field] of fields) {
    const entry = record[key]
    if (entry === undefined) {
      if (!field.isOptional && !isRequest) problems.push(`${path}.${key}: missing`)
      continue
    }
    if (field.isList) {
      if (!Array.isArray(entry)) problems.push(`${path}.${key}: not an array`)
      else {
        entry.forEach((one, index) =>
          problems.push(...check(one, field.type, `${path}.${key}[${index}]`, isRequest)),
        )
      }
    } else problems.push(...check(entry, field.type, `${path}.${key}`, isRequest))
  }
  return problems
}

const fixtures = Object.entries(fixtureFiles).map(([path, raw]) => {
  const name = path.slice(path.lastIndexOf('/') + 1).replace('.json', '')
  const parts = name.split('.')
  return {
    name,
    method: parts[0] ?? '',
    kind: parts[parts.length - 1] === 'request' ? 'request' : 'response',
    value: JSON.parse(raw) as unknown,
  }
})

describe('the fixtures of the contract', () => {
  it('cover the request and the response of every method', () => {
    const methods = [...interfaces.keys()]
      .filter((name) => name.endsWith('Request'))
      .map((name) => name.slice(0, -'Request'.length))
    expect(methods).toHaveLength(23)
    for (const method of methods) {
      expect(fixtures.some((one) => one.method === method && one.kind === 'request')).toBe(true)
      expect(fixtures.some((one) => one.method === method && one.kind === 'response')).toBe(true)
    }
  })

  it.each(fixtures)('$name is a message of the interfaces', ({ method, kind, value }) => {
    const typeName = method + (kind === 'request' ? 'Request' : 'Response')
    expect(check(value, typeName, typeName, kind === 'request')).toEqual([])
  })

  it('detects a field the interfaces do not declare', () => {
    expect(check({ path: 'a', extra: 1 }, 'ReadDeckRequest', 'ReadDeckRequest', true)).toEqual([
      'ReadDeckRequest.extra: not declared by ReadDeckRequest',
    ])
  })
})

describe('the client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the request fixture of a method and returns its response fixture', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })
    const { cardsService } = await import('./client')
    const request = fixtures.find((one) => one.name === 'ReadDeck.request')?.value
    const response = fixtures.find((one) => one.name === 'ReadDeck.response')?.value
    const requests: { url: string; body: unknown }[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string, init: { body: string }) => {
        requests.push({ url, body: JSON.parse(init.body) })
        return new Response(JSON.stringify(response), {
          headers: { 'content-type': 'application/json' },
        })
      }),
    )

    const answer = await cardsService.readDeck(request as { path: string })

    expect(requests).toEqual([{ url: 'http://numen.invalid/numen.flashcards/ReadDeck', body: request }])
    expect(answer).toEqual(response)
  })

  it('throws the API error of a failed method', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })
    const { cardsService } = await import('./client')
    const { ApiFailure } = await import('./failure')
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(JSON.stringify({ code: 'not_found', message: 'host: no such vault' }), {
            status: 404,
            headers: { 'content-type': 'application/json' },
          }),
      ),
    )

    const failure = await cardsService.readDeck({ path: 'a.md' }).catch((error: unknown) => error)

    expect(failure).toBeInstanceOf(ApiFailure)
    expect(failure).toMatchObject({ code: 'not_found', message: 'host: no such vault' })
  })
})
