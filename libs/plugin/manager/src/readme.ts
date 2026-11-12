/** The README.md of a plugin as its About tab draws it. */

const COMMENT_START = '<!--'
const COMMENT_END = '-->'
const LEADING_TITLE = /^\s*#[ \t]+\S[^\n]*(?:\n|$)/

/**
 * The text without its HTML comments. An unclosed comment runs to the end. A comment that the removal
 * of another one joins together is removed too.
 */
const removeComments = (raw: string): string => {
  let text = raw
  let start = text.indexOf(COMMENT_START)
  while (start !== -1) {
    const end = text.indexOf(COMMENT_END, start + COMMENT_START.length)
    text =
      end === -1
        ? text.slice(0, start)
        : text.slice(0, start) + text.slice(end + COMMENT_END.length)
    start = text.indexOf(COMMENT_START, Math.max(0, start - COMMENT_START.length + 1))
  }
  return text
}

/** The README without its HTML comments and without a leading `# Title`, which the header shows. */
export const getReadmeText = (raw: string): string =>
  removeComments(raw).replace(LEADING_TITLE, '').trim()
