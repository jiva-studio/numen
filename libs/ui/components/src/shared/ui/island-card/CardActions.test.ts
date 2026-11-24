/**
 * What the actions of a card header offer: every one is a named button, and the secondary ones are
 * also in the overflow menu.
 */
import { mount } from '@vue/test-utils'
import { KeyRound, Plus } from '@lucide/vue'
import { describe, expect, it } from 'vitest'
import CardActions from './CardActions.vue'
import type { CardActionItem } from './cardActionItem'

const ITEMS: readonly CardActionItem[] = [
  { id: 'words', label: 'Restore from words', icon: KeyRound },
  { id: 'create', label: 'Create', icon: Plus, isPrimary: true },
]

const mountActions = () =>
  mount(CardActions, { props: { items: ITEMS, moreLabel: 'More actions' }, attachTo: document.body })

describe('the card actions', () => {
  it('names every action for a button that shows only its icon', () => {
    const actions = mountActions()

    const names = actions.findAll('button').map((button) => button.attributes('aria-label'))

    expect(names).toEqual(['Restore from words', 'Create', 'More actions'])
    expect(actions.find('[title="Create"]').exists()).toBe(true)
  })

  it('emits the action that was pressed', async () => {
    const actions = mountActions()

    await actions.get('button[aria-label="Create"]').trigger('click')

    expect(actions.emitted('choose')).toEqual([['create']])
  })

  it('lists the secondary actions, and only them, in the overflow menu', async () => {
    const actions = mountActions()

    await actions.get('button[aria-label="More actions"]').trigger('click')

    const menu = document.body.querySelector('[role="menu"]')
    expect(menu?.textContent).toContain('Restore from words')
    expect(menu?.textContent).not.toContain('Create')
    actions.unmount()
  })

  it('draws no overflow menu where every action is primary', () => {
    const actions = mount(CardActions, {
      props: { items: [ITEMS[1]!], moreLabel: 'More actions' },
    })

    expect(actions.find('button[aria-label="More actions"]').exists()).toBe(false)
  })
})

describe('an action with choices', () => {
  const MENU: readonly CardActionItem[] = [
    {
      id: 'restore',
      label: 'Restore',
      icon: KeyRound,
      choices: [
        { id: 'words', label: 'From 12 words' },
        { id: 'typed', label: 'Type sheets' },
      ],
    },
    { id: 'create', label: 'Create', icon: Plus, isPrimary: true, isLabelKept: true },
  ]

  it('opens its choices, emits the one that was picked and stays out of the overflow menu', async () => {
    const actions = mount(CardActions, {
      props: { items: MENU, moreLabel: 'More actions' },
      attachTo: document.body,
    })

    expect(actions.find('button[aria-label="More actions"]').exists()).toBe(false)
    await actions.get('button[aria-label="Restore"]').trigger('click')
    const items = [...document.body.querySelectorAll('[role="menuitem"]')]
    expect(items.map((one) => one.textContent?.trim())).toEqual(['From 12 words', 'Type sheets'])
    ;(items[1] as HTMLElement).click()
    await actions.vm.$nextTick()
    await actions.vm.$nextTick()
    expect(document.activeElement).toBe(actions.get('button[aria-label="Restore"]').element)

    expect(actions.emitted('choose')).toEqual([['typed']])
    actions.unmount()
  })
})
