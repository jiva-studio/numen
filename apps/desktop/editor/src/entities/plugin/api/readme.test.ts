import { describe, expect, it } from 'vitest'
import { createAnsweredClients } from '@/testing/coreAnswers'
import { createReadmeReader } from './readme'

const createReader = (answer: () => object) =>
  createReadmeReader(
    createAnsweredClients({ pluginCatalog: { getCatalogReadme: answer } }).pluginCatalog,
  )

describe('createReadmeReader', () => {
  it('returns the text the core reads from the folder of the plugin', async () => {
    const read = createReader(() => ({ text: '# Hello' }))

    expect(await read('my-plugin')).toBe('# Hello')
  })

  it('asks the core for the plugin by its id', async () => {
    const asked: string[] = []
    const read = createReadmeReader(
      createAnsweredClients({
        pluginCatalog: {
          getCatalogReadme: (request: { id: string }) => {
            asked.push(request.id)
            return { text: '' }
          },
        },
      }).pluginCatalog,
    )

    await read('my-plugin')

    expect(asked).toStrictEqual(['my-plugin'])
  })

  it('returns an empty text where the plugin has no README', async () => {
    expect(await createReader(() => ({ text: '' }))('my-plugin')).toBe('')
  })

  it('returns an empty text where the core cannot answer', async () => {
    const read = createReader(() => {
      throw new Error('the core is closed')
    })

    expect(await read('my-plugin')).toBe('')
  })
})
