Feature: Manage vaults
  A person adds, renames and removes the vaults of the list.

  As a person who keeps notes in more than one vault
  I want to add, rename and remove vaults in the list of vaults
  So that the list holds the vaults I use

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

    Scenario: The open vault cannot be chosen
      Given the empty vault "Coastline"
      And the vault list has the following:
        | vault     |
        | Test      |
        | Coastline |
      When the person runs the command "Remove vault from list"
      Then the palette has the following rows:
        | list          | row       | disabled |
        | Search vaults | Coastline | False    |
        | Search vaults | Test      | True     |
