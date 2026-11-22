import { createApp } from 'vue'
import type { Field } from '../api/templates'
import FieldsForm from './FieldsForm.vue'

/** The values the person gave by label, and nothing when the person cancelled. */
type Answers = Readonly<Record<string, string>> | undefined

/** What asks the person for the fields of a template. */
export type AskFields = (title: string, fields: readonly Field[]) => Promise<Answers>

/** Opens the form of the fields over the window and resolves when the person creates or cancels. */
export const askFields: AskFields = (title, fields) =>
  new Promise((resolve) => {
    const host = document.createElement('div')
    document.body.append(host)
    const finish = (answers: Answers) => {
      app.unmount()
      host.remove()
      resolve(answers)
    }
    const app = createApp(FieldsForm, {
      title,
      fields,
      onSubmit: (values: Record<string, string>) => finish(values),
      onCancel: () => finish(undefined),
    })
    app.mount(host)
  })
