/** An hour and a minute of the day, as the field writes them. */
const CLOCK = /^([01]\d|2[0-3]):([0-5]\d)$/

export const isClockTime = (text: string): boolean => CLOCK.test(text)
