Feature: Plex
  A person sees the notes joined to the active note and joins more.

  As a person who keeps notes
  I want to see the notes joined to a note and to join more
  So that I can follow how my notes relate

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The plex links notes

    Scenario: A child note is made
      Given the vault has the following:
        | path             | text                                                                     |
        | notes/Entropy.md | # Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description. |
      And the person opens the note titled "Entropy"
      When the person runs the command "Create child note" with:
        | Name | Tributary |
      And the person runs the command "Show in plex" on the note tab titled "Entropy"
      Then the vault "Test" contains the note titled "Tributary"
      And the note titled "Entropy" links to the note titled "Tributary"
      And the plex has the child note titled "Tributary"

    Scenario: A note is linked by its title
      Given the vault has the following:
        | path           | text                        |
        | notes/Quay.md  | The quay is at the harbour. |
        | notes/Jetty.md | The jetty is at the quay.   |
      And the person opens the note titled "Quay"
      And the person opens the note titled "Jetty"
      And the person runs the command "Show in plex" on the note tab titled "Quay"
      When the person links the note titled "Quay" to the note titled "Jetty" in the plex
      Then the note titled "Quay" links to the note titled "Jetty"

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
      And the plex has the child note titled "Reef flat"

  Rule: The plex shows notes

    Scenario: A note is shown in the plex
      Given the vault has the following:
        | path             | text                                                                     |
        | notes/Entropy.md | # Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description. |
      When the person runs the command "Go to note" with:
        | Search notes | Entropy |
      And the person runs the command "Show in plex"
      Then the plex has the focus note titled "Entropy"

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

  Rule: A found note has actions

    Background:
      Given the vault has the following:
        | path          | text                        |
        | notes/Quay.md | The quay is at the harbour. |

    Scenario: A found note can be shown in the plex
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" contains the following actions:
        | action            |
        | Open note         |
        | Show in plex |

    Scenario: A found note has no plex action
      Given the person disables the plugin "Plex Graph"
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" has only the following actions:
        | action             |
        | Open note          |
        | Show in file tree  |
        | Create note “Quay” |
      And the search result "Quay" in the group "Names" does not contain the following actions:
        | action       |
        | Show in plex |
