Feature: Plex headings
  A person sees the headings of a note under its node in the plex.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path          | text                                      |
      | notes/Heat.md | # Heat\n\n## First law\n\nEnergy is kept. |
    And the application "editor" starts
    And the person opens the note titled "Heat"
    And the person runs the command "Show in plex" on the note tab titled "Heat"

  Scenario: The headings of a note are shown
    When the person hovers over the note titled "Heat" in the plex
    Then the plex has the heading "First law" under the note titled "Heat"

  Scenario: Headings are hidden
    Given the person hovers over the note titled "Heat" in the plex
    And the plex has the heading "First law" under the note titled "Heat"
    When the person runs the command "Plex Graph: Nest headings under notes" with:
      | Nest headings under notes | Off |
    And the person hovers over the note titled "Heat" in the plex
    Then the plex does not have the heading "First law" under the note titled "Heat"
