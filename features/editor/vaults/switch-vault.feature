Feature: Switch vault
  A person works in one vault at a time and switches to another.

  Background:
    Given the empty vault "Test"
    And the empty vault "Coastline"
    And the vault "Test" has the following:
      | path             | text      |
      | notes/Entropy.md | # Entropy |
    And the vault "Coastline" has the following:
      | path          | text                                      |
      | Lighthouse.md | # Lighthouse\n\nThe lamp turns all night. |
    And the application "editor" starts
    And the list of vaults has the vault "Coastline"

  Scenario: The window shows the other vault
    Given the person opens the file at "notes/Entropy.md"
    And the note tab at "notes/Entropy.md" is open
    When the person runs the command "Open vault"
    And the person picks "Coastline"
    Then the window has the vault "Coastline"
    And the note tab at "notes/Entropy.md" is not open
    And the Files tab titled "Files" has "Lighthouse.md"

  Scenario: The window shows the first vault again
    Given the person runs the command "Open vault"
    And the person picks "Coastline"
    And the window has the vault "Coastline"
    When the person runs the command "Open vault"
    And the person picks "Test"
    Then the window has the vault "Test"
    And the Files tab titled "Files" has "notes/Entropy.md"
