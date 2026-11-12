/**
 * What the search source says about how far it has indexed the vault, in its own words.
 *
 * The window shows the text as given; an installation without a search source gives none.
 */

/** The vault's index as the search source reports it. */
export interface IndexCoverage {
  readonly text: string
}
