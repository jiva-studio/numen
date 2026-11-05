/** A cell of a table row: a bar or a line break in the text would split the row. */
export const cell = (text) =>
  text
    .replace(/\s*\n\s*/g, ' ')
    .replace(/\\/g, '\\\\')
    .replace(/\|/g, '\\|')
