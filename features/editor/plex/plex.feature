Feature: Plex
  A person sees the notes joined to the note in front and joins more.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The plex links notes

    Scenario: A child note is made
      Given the vault has the following:
        | path             | text                                                                     |
        | notes/Entropy.md | # Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description. |
      And the person opens the note titled "Entropy"
      When the person runs the command "Create child note"
      And the person names it "Tributary"
      And the person runs the command "Show note in plex" on the note tab titled "Entropy"
      Then the vault "Test" contains the note titled "Tributary"
      And the note titled "Entropy" links to the note titled "Tributary"
      And the plex has the note titled "Tributary" as the "child"

    Scenario: A note is linked by its title
      Given the vault has the following:
        | path           | text                        |
        | notes/Quay.md  | The quay is at the harbour. |
        | notes/Jetty.md | The jetty is at the quay.   |
      And the person opens the note titled "Quay"
      And the person opens the note titled "Jetty"
      And the person runs the command "Show note in plex" on the note tab titled "Quay"
      When the person links the note titled "Quay" to the note titled "Jetty" in the plex
      Then the note titled "Quay" links to the note titled "Jetty"

    Scenario: A child note is made from the file tree
      Given the vault has the following:
        | path            | text                |
        | notes/Lagoon.md | The lagoon is calm. |
      When the person runs the command "Create child note" on the file at "notes/Lagoon.md"
      And the person names it "Reef flat"
      And the person opens the file at "notes/Lagoon.md"
      And the person runs the command "Show note in plex"
      Then the vault "Test" contains the note titled "Reef flat"
      And the note titled "Lagoon" links to the note titled "Reef flat"
      And the plex has the note titled "Reef flat" as the "child"

  Rule: The plex shows notes

    Scenario: A note is shown in the plex
      Given the vault has the following:
        | path             | text                                                                     |
        | notes/Entropy.md | # Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description. |
      When the person runs the command "Go to note"
      And the person picks "Entropy"
      And the person runs the command "Show note in plex"
      Then the plex has the note titled "Entropy" as the "focus"

    Scenario: A linked tab follows a click
      Given the vault has the following:
        | path                    | text                                                                                                          |
        | notes/Weir notes.md     | Water over the weir.                                                                                          |
        | notes/Entropy.md        | # Entropy\n\nA measure of uncertainty.                                                                        |
        | notes/Thermodynamics.md | ---\nlinks:\n  - to: "[[notes/Entropy]]"\n    role: child\n---\n\n# Thermodynamics\n\nThe study of energy. |
      And the person opens the note titled "Thermodynamics"
      And the person opens the note titled "Weir notes"
      And the person runs the command "Show note in plex" on the note tab titled "Thermodynamics"
      And the person runs the command "Link with tab" on the Plex tab titled "Thermodynamics"
      And the person picks "Weir notes"
      When the person clicks the note titled "Entropy" in the plex
      Then the tab at "notes/Entropy.md" is active
      And the tab titled "Weir notes" is not open

  Rule: A found note has actions

    Background:
      Given the vault has the following:
        | path          | text                        |
        | notes/Quay.md | The quay is at the harbour. |

    Scenario: The plex is an action of a found note
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" has the following actions:
        | action            |
        | Open note         |
        | Show note in plex |

    Scenario: A found note without the plex is only opened
      Given the person disables the plugin "Plex Graph"
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" has the following actions:
        | action    |
        | Open note |
      When the person chooses the search result "Quay" in the group "Names"
      Then the tab at "notes/Quay.md" is active
