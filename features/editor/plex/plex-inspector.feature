Feature: Plex link inspector
  A person changes a link between two notes in the plex.

  As a person who keeps notes
  I want to change the link between two notes
  So that the link says what the notes are to each other

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text      |
      | notes/Harbour.md | # Harbour |
      | notes/Quay.md    | # Quay    |
    And the notes have the following links:
      | from    | to   | role  |
      | Harbour | Quay | child |
    And the application "editor" starts
    And the person opens the note titled "Harbour"
    And the person runs the command "Show in plex" on the note tab titled "Harbour"

  Rule: The link inspector changes the arrows and the description of a link

    Scenario Outline: An arrow is added to a link
      Given the plex draws the line from "Harbour" to "Quay" without an arrow
      When the person changes the link from "Harbour" to "Quay" in the plex with:
        | Arrow | <arrow> |
      Then the plex draws the line from "Harbour" to "Quay" with an arrow to "<end>"

      Examples:
        | arrow            | end     |
        | Arrow to Quay    | Quay    |
        | Arrow to Harbour | Harbour |

    Scenario: An arrow is removed from a link
      Given the person changes the link from "Harbour" to "Quay" in the plex with:
        | Arrow | Arrow to Quay |
      And the plex draws the line from "Harbour" to "Quay" with an arrow to "Quay"
      When the person changes the link from "Harbour" to "Quay" in the plex with:
        | Arrow | No arrow |
      Then the plex draws the line from "Harbour" to "Quay" without an arrow

    Scenario: A description is added to a link
      When the person changes the link from "Harbour" to "Quay" in the plex with:
        | Description | Daily ferry |
      Then the vault "Test" has the following links:
        | from    | to   | role  | description |
        | Harbour | Quay | child | Daily ferry |

    Scenario: A cancelled change leaves the links unchanged
      When the person cancels the change of the link from "Harbour" to "Quay" in the plex with:
        | Arrow       | Arrow to Quay |
        | Description | Daily ferry   |
      Then the vault "Test" has the following links:
        | from    | to   | role  |
        | Harbour | Quay | child |
      And the plex draws the line from "Harbour" to "Quay" without an arrow

    Scenario: A reverse link is added
      When the person adds the reverse link from "Quay" to "Harbour" in the plex
      Then the note titled "Harbour" links to the note titled "Quay"
      And the note titled "Quay" links to the note titled "Harbour"

    Scenario: One direction of a link is removed
      Given the person adds the reverse link from "Quay" to "Harbour" in the plex
      When the person removes the link from "Quay" to "Harbour" in the plex
      Then the note titled "Harbour" links to the note titled "Quay"
      And the note titled "Quay" does not link to the note titled "Harbour"

    Scenario: A link is removed
      When the person removes the link between "Harbour" and "Quay" in the plex
      Then the note titled "Harbour" does not link to the note titled "Quay"
