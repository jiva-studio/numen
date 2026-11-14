import { describe, expect, it } from 'vitest'
import { createSavedConversation, parseSavedConversation } from './saved'

describe('the conversation an agent tab saves', () => {
  it('keeps the question and the settled prompts and responses', () => {
    const saved = createSavedConversation('and then?', [
      { id: '1', voice: 'prompt', text: 'What is a heron?' },
      { id: '2', voice: 'doing', text: 'Reading Birds.md' },
      { id: '3', voice: 'response', text: 'A wading bird.' },
      { id: '4', voice: 'response', text: 'It', state: 'arriving' },
    ])

    expect(saved).toStrictEqual({
      question: 'and then?',
      turns: [
        { id: '1', voice: 'prompt', text: 'What is a heron?' },
        { id: '3', voice: 'response', text: 'A wading bird.' },
      ],
      model: null,
    })
  })

  it('reads back as saved, and as empty where the value is no saved conversation', () => {
    const saved = createSavedConversation('', [{ id: '1', voice: 'prompt', text: 'Why?' }], {
      connectionId: 'c-lantern',
      modelId: 'c-meadow|clover',
    })

    expect(parseSavedConversation(JSON.parse(JSON.stringify(saved)))).toStrictEqual(saved)
    expect(parseSavedConversation(undefined)).toStrictEqual({
      question: '',
      turns: [],
      model: null,
    })
    expect(
      parseSavedConversation({ turns: [{ id: 1, voice: 'x' }], model: { connectionId: 7 } }),
    ).toStrictEqual({ question: '', turns: [], model: null })
  })
})
