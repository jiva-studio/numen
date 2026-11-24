const GROUP = 3

/** A confirmation code in groups of three digits, as it reads aloud: 482 913. */
export const getCodeText = (code: string): string =>
  code.replace(new RegExp(`(\\d{${GROUP}})(?=\\d)`, 'g'), '$1 ')
