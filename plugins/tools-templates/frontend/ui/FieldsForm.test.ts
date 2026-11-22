import { afterEach, describe, expect, it } from 'vitest'
import { createApp, type App } from 'vue'
import type { Field } from '../api/templates'
import FieldsForm from './FieldsForm.vue'

const FIELDS: readonly Field[] = [
  { kind: 'text', label: 'Client', options: [], default: 'Acme' },
  { kind: 'select', label: 'Kind', options: ['call', 'visit'], default: 'call' },
]

let mounted: { app: App; host: HTMLElement } | undefined

afterEach(() => {
  mounted?.app.unmount()
  mounted?.host.remove()
  mounted = undefined
})

const open = (fields: readonly Field[] = FIELDS) => {
  const submitted: Record<string, string>[] = []
  let cancelled = 0
  const host = document.createElement('div')
  document.body.append(host)
  const app = createApp(FieldsForm, {
    title: 'Meeting',
    fields,
    onSubmit: (values: Record<string, string>) => submitted.push(values),
    onCancel: () => (cancelled += 1),
  })
  app.mount(host)
  mounted = { app, host }
  return { host, submitted, getCancelled: () => cancelled }
}

const inputOf = (host: HTMLElement, label: string): HTMLInputElement => {
  const input = host.querySelector<HTMLInputElement>(`input[aria-label="${label}"]`)
  if (!input) throw new Error(`no input ${label}`)
  return input
}

const type = (input: HTMLInputElement, text: string) => {
  input.value = text
  input.dispatchEvent(new Event('input'))
}

describe('the form of the fields of a template', () => {
  it('names the template and asks each text field with its default', () => {
    const { host } = open()

    expect(host.querySelector('[role="dialog"]')?.getAttribute('aria-label')).toBe('Meeting')
    expect(inputOf(host, 'Client').value).toBe('Acme')
  })

  it('submits the defaults of the fields as they are', () => {
    const { host, submitted } = open()

    host.querySelector('form')?.dispatchEvent(new Event('submit', { cancelable: true }))

    expect(submitted).toEqual([{ Client: 'Acme', Kind: 'call' }])
  })

  it('submits what was typed', async () => {
    const { host, submitted } = open()

    type(inputOf(host, 'Client'), 'Globex')
    host.querySelector('form')?.dispatchEvent(new Event('submit', { cancelable: true }))

    expect(submitted).toEqual([{ Client: 'Globex', Kind: 'call' }])
  })

  it('cancels on the Cancel button and on Escape', () => {
    const { host, getCancelled } = open()

    host.querySelector<HTMLButtonElement>('button[data-role="cancel"]')?.click()
    host
      .querySelector('[role="dialog"]')
      ?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))

    expect(getCancelled()).toBe(2)
  })

  it('draws a select field as a select', () => {
    const { host } = open()

    expect(host.querySelector('[data-field="Kind"]')).not.toBeNull()
    expect(host.querySelector('input[aria-label="Kind"]')).toBeNull()
  })
})
