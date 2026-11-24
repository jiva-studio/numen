const FORMAT = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' })

/** The moment a link stops working, in the language of the system. */
export const getExpiryText = (expiresAt: string): string => FORMAT.format(new Date(expiresAt))
