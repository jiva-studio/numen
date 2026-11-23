import type { PickerHost } from '@numen/plugin-sdk'
import type { CreateNoteResponse } from '../api/templates'
import type { TemplatesApi } from '../api/wire'
import { getTitle } from '../lib/groupTemplates'
import { collectFieldValues } from './collectFieldValues'

/** What making a note from a template came to. */
export type Outcome =
  | { readonly kind: 'made'; readonly answer: CreateNoteResponse }
  | { readonly kind: 'cancelled' }
  | { readonly kind: 'failed'; readonly message: string }

const NAME_REFUSAL_CODES: ReadonlySet<string> = new Set(['already_exists', 'invalid_argument'])

/** What makes the note: the backend that lists the questions and makes it, and the palette that asks them. */
export interface CreateFromTemplateDeps {
  readonly api: TemplatesApi
  readonly picker: Pick<PickerHost, 'promptText' | 'selectOption'> | undefined
}

const createFailedOutcome = (message: string): Outcome => ({ kind: 'failed', message })

/**
 * Asks the name of the note and then the fields of the template in the palette, and makes the note.
 * A name the backend refuses is asked again, with the reason in the question, until it is taken or
 * the person dismisses the question.
 */
export async function createFromTemplate(
  deps: CreateFromTemplateDeps,
  template: string,
  folder: string,
): Promise<Outcome> {
  const { api, picker } = deps
  const asked = await api.listFields({ template, folder })
  if (!asked.ok) return createFailedOutcome(asked.error.message)
  if (!picker) return createFailedOutcome('the window cannot ask the name of the note')
  const title = getTitle(template)
  const askName = (value: string, refusal: string): Promise<string | undefined> =>
    picker.promptText({ title: `${title} · Name${refusal ? ` · ${refusal}` : ''}`, value })

  let name = await askName(asked.value.name, '')
  if (name === undefined) return { kind: 'cancelled' }
  const values = await collectFieldValues(picker, title, asked.value.fields)
  if (!values) return { kind: 'cancelled' }
  for (;;) {
    const answer = await api.createNote({ template, folder, name, values, etag: asked.value.etag })
    if (answer.ok) return { kind: 'made', answer: answer.value }
    if (!NAME_REFUSAL_CODES.has(answer.error.code)) return createFailedOutcome(answer.error.message)
    name = await askName(name, answer.error.message)
    if (name === undefined) return { kind: 'cancelled' }
  }
}
