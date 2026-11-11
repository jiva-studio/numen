Feature: Manage vaults
  A person adds, renames and removes the vaults of the list.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A vault is added and renamed

    Scenario: A vault is created from a folder
      Given the empty vault "Coastline"
      When the person runs the command "Create vault" with:
        | Folder | Coastline |
      Then the vault list has the following:
        | vault     |
        | Test      |
        | Coastline |
      And the window has the vault "Coastline"

    Scenario: A vault is renamed
      When the person runs the command "Rename vault" with:
        | Name | Harbor |
      Then the window has the vault "Harbor"

  Rule: A vault is removed from the list

    Scenario: A vault is removed
      Given the empty vault "Coastline"
      And the vault list has the following:
        | vault     |
        | Test      |
        | Coastline |
      When the person runs the command "Remove vault from list" with:
        | Search vaults | Coastline |
        | Confirm       | Remove    |
      Then the vault list has the following:
        | vault |
        | Test  |

    Scenario: The only vault is not removed
      Given the window has the vault "Test"
      When the person runs the command "Remove vault from list" with:
        | Search vaults | Test |
        | Confirm       | Remove |
      Then the window shows the message "the open vault cannot be removed"
      And the window has the vault "Test"
