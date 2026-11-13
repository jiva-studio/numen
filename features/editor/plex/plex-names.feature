Feature: Plex names
  A person sees the names of the notes related to the related notes.

  As a person who keeps notes
  I want to see what the related notes are related to
  So that I can go on to the next note without a click on each

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path                    | text             |
      | notes/Orchard plan.md   | # Orchard plan   |
      | notes/Pear rows.md      | # Pear rows      |
      | notes/Rootstock list.md | # Rootstock list |
    And the notes have the following links:
      | from         | to             | role  |
      | Orchard plan | Pear rows      | child |
      | Pear rows    | Rootstock list | child |
    And the application "editor" starts
    And the person opens the note titled "Orchard plan"
    And the person runs the command "Show in plex" on the note tab titled "Orchard plan"

  Scenario: Names are shown under a note
    Then the plex shows the following names:
      | note      | name           |
      | Pear rows | Rootstock list |
    And the plex does not show the child note titled "Rootstock list"

  Scenario: A name moves the focus
    When the person rests the pointer on the note titled "Pear rows" in the plex
    And the person clicks the name "Rootstock list" under the note titled "Pear rows"
    Then the plex shows the focus note titled "Rootstock list"

  Scenario: Names are hidden
    Given the plex shows the following names:
      | note      | name           |
      | Pear rows | Rootstock list |
    When the person sets the setting "Show notes related to related notes" of the plugin "Plex Graph" to "Off"
    Then the plex shows no names under the note titled "Pear rows"
