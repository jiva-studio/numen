Feature: Plex
  A person sees the notes joined to the active note and joins more.

  As a person who keeps notes
  I want to see the notes joined to a note and to join more
  So that I can follow how my notes relate

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A child note made from the file tree is a child in the plex

    Scenario: A child note is made from the file tree
      Given the vault has the following:
        | path            | text                |
        | notes/Lagoon.md | The lagoon is calm. |
      When the person runs the command "Create child note" on the file at "notes/Lagoon.md" with:
        | Name | Reef flat |
      And the person opens the file at "notes/Lagoon.md"
      And the person runs the command "Show in plex"
      Then the vault "Test" contains the note titled "Reef flat"
      And the note titled "Lagoon" links to the note titled "Reef flat"
      And the plex shows the child note titled "Reef flat"

  Rule: A linked tab follows the notes of the plex

    Scenario: A linked tab follows a click
      Given the vault has the following:
        | path                    | text                                                                                                          |
        | notes/Weir notes.md     | Water over the weir.                                                                                          |
        | notes/Entropy.md        | # Entropy\n\nA measure of uncertainty.                                                                        |
        | notes/Thermodynamics.md | ---\nlinks:\n  - to: "[[notes/Entropy]]"\n    role: child\n---\n\n# Thermodynamics\n\nThe study of energy. |
      And the person opens the note titled "Thermodynamics"
      And the person opens the note titled "Weir notes"
      And the person runs the command "Show in plex" on the note tab titled "Thermodynamics"
      And the person runs the command "Link with tab" on the Plex tab titled "Thermodynamics" with:
        | Search | Weir notes |
      When the person clicks the note titled "Entropy" in the plex
      Then the window has the following tabs:
        | kind | path             |
        | note | notes/Entropy.md |
      And the tab titled "Weir notes" is not open

  Rule: A note without links is shown alone

    Scenario: A note without links shows only the focus
      Given the vault has the following:
        | path            | text                |
        | notes/Lagoon.md | The lagoon is calm. |
      And the person opens the note titled "Lagoon"
      When the person runs the command "Show in plex" on the note tab titled "Lagoon"
      Then the plex shows only the focus note titled "Lagoon"
