/** The README.md of a plugin as its About tab draws it. */

const HTML_COMMENT = /<!--[\s\S]*?-->/g
const LEADING_TITLE = /^\s*#[ \t]+\S[^\n]*(?:\n|$)/

/** The README without its HTML comments and without a leading `# Title`, which the header shows. */
export const getReadmeText = (raw: string): string =>
  raw.replace(HTML_COMMENT, '').replace(LEADING_TITLE, '').trim()
