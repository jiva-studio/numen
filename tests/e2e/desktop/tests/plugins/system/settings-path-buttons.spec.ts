import { mkdir, writeFile } from 'node:fs/promises'
import { dirname } from 'node:path'
import { test, expect, type Locator, type Page } from '../../support/test.js'
import { writeSetting } from '../../support/vault.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

const SINGLE = { id: 'md.numen.agents.antigravity', name: 'Antigravity Agent' }
const MANY = { id: 'md.numen.agents.acp', name: 'ACP Agents' }

const createExecutable = async (name: string): Promise<string> => {
  const path = test.info().outputPath(name, 'bin', 'agent')
  await mkdir(dirname(path), { recursive: true })
  await writeFile(path, '#!/bin/sh\n', { mode: 0o755 })
  return path
}

/**
 * How a button is drawn against the tokens it takes: its border is the field border and its
 * background the field background, and the border is not the colour of its text, which is what an
 * undeclared border token falls back to.
 */
const readButtonLook = (button: Locator) =>
  button.evaluate((one) => {
    const resolve = (property: 'borderTopColor' | 'backgroundColor', token: string) => {
      const probe = document.createElement('div')
      probe.style[property] = `var(${token})`
      one.parentElement?.append(probe)
      const value = getComputedStyle(probe)[property]
      probe.remove()
      return value
    }
    const style = getComputedStyle(one)
    return {
      border: style.borderTopColor,
      borderToken: resolve('borderTopColor', '--numen-field-border'),
      text: style.color,
      background: style.backgroundColor,
      backgroundToken: resolve('backgroundColor', '--numen-field-bg'),
      borderWidth: style.borderTopWidth,
    }
  })

const expectThemedButton = async (button: Locator): Promise<void> => {
  const look = await readButtonLook(button)
  expect(look.borderWidth).not.toBe('0px')
  expect(look.border).toBe(look.borderToken)
  expect(look.border).not.toBe(look.text)
  expect(look.background).toBe(look.backgroundToken)
}

const openPlugin = async (page: Page, name: string): Promise<Locator> => {
  await openWorkspace(page)
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: new RegExp(name) }).click()
  await expect(settings.locator('.plugin-detail__title')).toContainText(name)
  return settings
}

for (const mode of ['light', 'dark'] as const) {
  test.describe(`The buttons of a path setting in the ${mode} mode`, () => {
    test.beforeEach(async ({ request }) => {
      await writeSetting(request, ['appearance', 'mode'], mode)
    })

    test.afterEach(async ({ request }) => {
      await writeSetting(request, ['appearance', 'mode'], 'system')
      await writeSetting(request, ['plugins', SINGLE.id, 'executable_path'], '')
      await writeSetting(request, ['plugins', MANY.id, 'executable_path'], '')
    })

    test('Choose and Clear are drawn as the buttons of the kit', async ({ page, request }) => {
      await writeSetting(
        request,
        ['plugins', SINGLE.id, 'executable_path'],
        await createExecutable('one'),
      )
      const settings = await openPlugin(page, SINGLE.name)
      const field = settings.getByRole('group', { name: 'Executable Path' })

      await expectThemedButton(field.getByRole('button', { name: 'Choose…' }))
      await expectThemedButton(field.getByRole('button', { name: 'Clear' }))
    })

    test('Add and each Clear are drawn as the buttons of the kit', async ({ page, request }) => {
      const paths = [await createExecutable('one'), await createExecutable('two')]
      await writeSetting(request, ['plugins', MANY.id, 'executable_path'], paths.join('\n'))
      const settings = await openPlugin(page, MANY.name)
      const field = settings.getByRole('group', { name: 'Executable Paths' })

      await expectThemedButton(field.getByRole('button', { name: 'Add…' }))
      await expect(field.getByRole('button', { name: 'Clear' })).toHaveCount(2)
      await expectThemedButton(field.getByRole('button', { name: 'Clear' }).first())
      await expectThemedButton(field.getByRole('button', { name: 'Clear' }).last())
    })
  })
}
