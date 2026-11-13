/**
 * A PDF tab whose document cannot be opened draws the message the failure is reported with in a fallback panel
 * where the pages would be. The tab's own reader runs over a made-up port that refuses the document.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import type { Documents } from '../types'

vi.mock('../reader', () => ({
  Reader: defineComponent({
    name: 'Reader',
    setup:
      (_props, { slots }) =>
      () =>
        h('div', { class: 'reader-stub' }, slots.silence?.()),
  }),
}))

vi.mock('@numen/ui', () => ({
  LoadingIndicator: () => h('span'),
}))

const PATH = 'library/Corrupted.pdf'

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

const failure = new Error('the file is not a PDF')

const refusing: Documents = {
  getDocumentLayout: async () => {
    throw failure
  },
  getPageUrl: () => '',
  getHighlights: async () => [],
}

describe('a PDF tab over a document that cannot be opened', () => {
  it('draws the message the failure is reported with in the fallback panel', async () => {
    const state = reactive({
      path: PATH,
      title: 'Corrupted',
      selector: null,
      ranges: null,
      savePlace: vi.fn(),
      reportPlace: vi.fn(),
    })
    const tab = mount(DocumentTab, { props: { state, documents: refusing } })
    await flush()
    await flush()

    expect(tab.find('.fallback-panel').exists()).toBe(true)
    expect(tab.find('.fallback-message').text()).toBe(formatErrorMessage(failure))
    tab.unmount()
  })
})
