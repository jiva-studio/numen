Feature: File tree
  A person moves, makes, renames and removes the files of a vault in the file tree.

  Background:
    Given the application "editor" starts

  Rule: The file tree changes the files of the vault

    Scenario: A note is moved to a folder and its links follow
      Given the vault "Test" has the folder at "daily"
      And the person has the note titled "Quay" with "The quay is at the harbour."
      And the person has the note titled "Jetty" with "See [[Quay]]."
      When the person moves the file at "notes/Quay.md" to the folder at "daily"
      Then the tab at "daily/Quay.md" is open
      And the tab at "notes/Quay.md" is not open
      And the note titled "Jetty" links to the note at "daily/Quay.md"

    Scenario: A folder is made in a folder
      Given the vault "Test" has the folder at "Saltworks"
      When the person runs the command "Create folder" on the folder at "Saltworks"
      And the person enters "Brine tanks"
      Then the vault "Test" has the folder at "Saltworks/Brine tanks"

    Scenario: A note is renamed and its tab follows
      Given the person has the note titled "Brine pump" with "The pump runs at night."
      When the person runs the command "Rename file" on the file at "notes/Brine pump.md"
      And the person enters "Brine pumps.md"
      Then the tab at "notes/Brine pumps.md" is open
      And the tab at "notes/Brine pump.md" is not open

    Scenario: The tab of a removed note is closed
      Given the person has the note titled "Mill race" with "The race feeds the wheel."
      When the person runs the command "Move note to trash" on the file at "notes/Mill race.md"
      Then the tab at "notes/Mill race.md" is not open
      And the vault "Test" does not have the file at "notes/Mill race.md"

  Rule: The file tree has the files of the vault

    Scenario: A folder is opened in the file tree
      When the person opens the folder at "cards"
      Then the tab titled "Files" has "cards/Animals.md"

    Scenario: A note is shown in the file tree
      Given the person opens the file at "notes/Entropy.md"
      When the person runs the command "Show note in file tree"
      Then the tab titled "Files" is active
      And the tab titled "Files" has "notes/Entropy.md"
