/** An hour and a minute of the day, as the field writes them. */
const CLOCK = /^([01]\d|2[0-3]):([0-5]\d)$/

/**
 * Whether these are an hour and a minute on the clock on the wall, written as
 * `04:00`. A field left empty stands at no hour and is not one.
 */
export const onTheClock = (said: string): boolean => CLOCK.test(said)
