Feature: Plex
  A person sees the notes joined to the note in front and joins more.

  Background:
    Given the application "editor" starts
    And the person opens the file at "notes/Entropy.md"

  Rule: A note is linked to the note in front

    Scenario: A child note is linked to the note in front
      When the person runs the command "Create child note"
      And the person names it "Tributary"
      And the person runs the command "Show note in plex" on the tab titled "Entropy"
      Then the vault "Test" contains the note titled "Tributary"
      And the note titled "Entropy" links to the note titled "Tributary"
      And the plex has the note titled "Tributary" as the "child"

    Scenario: A note is linked to a note found by its title
      Given the vault has the following:
        | path          | text                        |
        | notes/Quay.md | The quay is at the harbour. |
      And the person opens the note titled "Quay"
      And the vault has the following:
        | path           | text                      |
        | notes/Jetty.md | The jetty is at the quay. |
      And the person opens the note titled "Jetty"
      And the person runs the command "Show note in plex" on the tab titled "Quay"
      When the person links the note titled "Quay" to the note titled "Jetty" in the plex
      Then the note titled "Quay" links to the note titled "Jetty"

  Rule: A note is shown in the plex

    Scenario: A note found by its title is the focus of the plex
      When the person runs the command "Go to note"
      And the person picks "Entropy"
      And the person runs the command "Show note in plex"
      Then the plex has the note titled "Entropy" as the "focus"

    Scenario: A tab linked to the plex opens the note clicked in it
      Given the vault has the following:
        | path                | text                 |
        | notes/Weir notes.md | Water over the weir. |
      And the person opens the note titled "Weir notes"
      And the person runs the command "Link with tab" on the tab titled "Thermodynamics"
      And the person picks "Weir notes"
      When the person clicks the note titled "Entropy" in the plex
      Then the tab at "notes/Entropy.md" is active
      And the tab titled "Weir notes" is not open
