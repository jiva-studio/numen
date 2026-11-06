import { describe, expect, it } from 'vitest'
import type { SlotItem } from '@numen/plugin-sdk'
import { createBarMenu } from './menu'
import { command, separator, submenu, view } from '../testing/entries'

const texts = (items: readonly { text: string }[]) => items.map((one) => one.text)

describe('createBarMenu', () => {
  it('lists the commands in order under their titles, and runs each by its command', () => {
    const menu = createBarMenu([command('chart.open'), command('map.open')])

    expect(texts(menu.items)).toStrictEqual(['Open chart.open', 'Open map.open'])
    expect(menu.items.map((one) => menu.commands.get(one.id))).toStrictEqual([
      'org.example.tide:chart.open',
      'org.example.tide:map.open',
    ])
  })

  it('puts the entries on each side of a separator in two groups', () => {
    const menu = createBarMenu([command('a'), separator(1), command('b'), command('c')])

    const [a, b, c] = menu.items
    expect(a?.group).not.toBe(b?.group)
    expect(b?.group).toBe(c?.group)
  })

  it('draws a submenu as a row whose children are its entries', () => {
    const menu = createBarMenu([
      submenu('fileTree.new', [command('note.create'), separator(2), command('deck.create')]),
    ])

    const [row] = menu.items
    expect(row?.text).toBe('New')
    expect(texts(row?.children ?? [])).toStrictEqual(['Open note.create', 'Open deck.create'])
    expect(menu.commands.get(row!.children![1]!.id)).toBe('org.example.tide:deck.create')
    expect(menu.commands.has(row!.id)).toBe(false)
  })

  it('leaves out a view, which is never drawn in a menu', () => {
    const items: SlotItem[] = [view('tasks'), command('a')]

    expect(texts(createBarMenu(items).items)).toStrictEqual(['Open a'])
  })

  it('gives each row an id of its own when a command is placed twice', () => {
    const menu = createBarMenu([command('a'), separator(1), command('a')])

    expect(new Set(menu.items.map((one) => one.id)).size).toBe(2)
  })

  it('keeps the icon of each command under the id of its row', () => {
    const menu = createBarMenu([command('a')])

    expect(menu.icons.get(menu.items[0]!.id)).toBeDefined()
  })
})
