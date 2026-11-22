import type { PickerHost } from '@numen/plugin-sdk'
import type { Field } from '../api/templates'

type Palette = Pick<PickerHost, 'ask' | 'choose'>

/** The answers by label, in the order of the fields, or undefined once the person dismisses a question. */
export const askFields = async (
  palette: Palette,
  title: string,
  fields: readonly Field[],
): Promise<Readonly<Record<string, string>> | undefined> => {
  const values: Record<string, string> = {}
  for (const [index, field] of fields.entries()) {
    const question = `${title} · ${index + 1}/${fields.length} ${field.label}`
    const answer =
      field.kind === 'select'
        ? await palette.choose({
            title: question,
            options: [field.default, ...field.options.filter((one) => one !== field.default)]
              .filter((one) => one !== '')
              .map((one) => ({ value: one, title: one })),
          })
        : await palette.ask({ title: question, value: field.default })
    if (answer === undefined) return undefined
    values[field.label] = answer
  }
  return values
}
