Feature: File tree
  A person moves, makes, renames and removes the files of a vault in the file tree.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The file tree changes files

    Scenario: A note is moved with its links
      Given the vault has the following:
        | path           | text                        |
        | daily/         |                             |
        | notes/Quay.md  | The quay is at the harbour. |
        | notes/Jetty.md | See [[Quay]].               |
      And the person opens the note titled "Quay"
      And the person opens the note titled "Jetty"
      When the person moves the file at "notes/Quay.md" to the folder at "daily"
      Then the tab at "daily/Quay.md" is open
      And the tab at "notes/Quay.md" is not open
      And the note titled "Jetty" links to the note at "daily/Quay.md"

    Scenario: A folder is made in a folder
      Given the vault has the following:
        | path       | text |
        | Saltworks/ |      |
      When the person runs the command "Create folder" on the folder at "Saltworks" with:
        | Name | Brine tanks |
      Then the vault contains the folder at "Saltworks/Brine tanks"

    Scenario: A folder is made with the palette command
      When the person runs the command "Create folder" with:
        | Name | Saltworks |
      Then the vault contains the folder at "Saltworks"

    Scenario: A note is renamed
      Given the vault has the following:
        | path                | text                    |
        | notes/Brine pump.md | The pump runs at night. |
      And the person opens the note titled "Brine pump"
      When the person runs the command "Rename" on the file at "notes/Brine pump.md" with:
        | Name | Brine pumps.md |
      Then the tab at "notes/Brine pumps.md" is open
      And the tab at "notes/Brine pump.md" is not open

  Rule: The file tree shows files

    Scenario: A folder is opened
      Given the vault has the following:
        | path             | text                      |
        | cards/Animals.md | A heron is a wading bird. |
      When the person opens the folder at "cards"
      Then the tab titled "Files" has "cards/Animals.md"

    Scenario: A note is shown
      Given the vault has the following:
        | path             | text      |
        | notes/Entropy.md | # Entropy |
      And the person opens the file at "notes/Entropy.md"
      When the person runs the command "Show note in file tree"
      Then the tab titled "Files" is active
      And the tab titled "Files" has "notes/Entropy.md"
