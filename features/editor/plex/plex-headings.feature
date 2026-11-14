Feature: Plex headings
  A person sees the headings of a note under its node in the plex.

  As a person who keeps notes
  I want to see the headings of a note in the plex
  So that I can see what a note holds without opening it

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path          | text                                      |
      | notes/Heat.md | # Heat\n\n## First law\n\nEnergy is kept. |
      | notes/Work.md | # Work\n\n## Joule\n\nOne joule is a newton metre. |
    And the notes have the following links:
      | from | to   | role  |
      | Heat | Work | child |
    And the application "editor" starts
    And the person opens the note titled "Heat"
    And the person runs the command "Show in plex" on the note tab titled "Heat"

  Scenario: The headings of a note open when the pointer rests on it
    When the person rests the pointer on the note titled "Heat" in the plex
    Then the plex shows the heading "First law" under the note titled "Heat"

  Scenario: One list of headings is open at a time
    Given the person rests the pointer on the note titled "Heat" in the plex
    And the plex shows the heading "First law" under the note titled "Heat"
    When the person rests the pointer on the note titled "Work" in the plex
    Then the plex shows the heading "Joule" under the note titled "Work"
    And the plex does not show the heading "First law" under the note titled "Heat"

  Scenario: A heading opens its note
    When the person enters the heading "Joule" under the note titled "Work" in the plex
    Then the note tab at "notes/Work.md" is active
    And the cursor of the note tab titled "Work" is on the line "## Joule"

  Scenario: Headings are hidden
    Given the person rests the pointer on the note titled "Heat" in the plex
    And the plex shows the heading "First law" under the note titled "Heat"
    When the person sets the setting "Nest headings under notes" of the plugin "Plex Graph" to "Off"
    And the person rests the pointer on the note titled "Heat" in the plex
    Then the plex does not show the heading "First law" under the note titled "Heat"
