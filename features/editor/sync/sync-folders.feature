@sync-server
Feature: Sync a moved folder between computers

  Numen Sync sends a folder that moves as one change.

  As a person who keeps order in a vault
  I want a folder I rename to be renamed on the other computer
  So that both computers show the same folders

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: A moved folder arrives whole

    Scenario: A renamed folder is renamed on the second computer
      Given the person writes the note "Food/Shopping list" with the text "milk" in the vault "Test"
      And the person adds the file "Food/Menu.pdf" to the vault "Test"
      And the second computer has the note "Food/Shopping list" with the text "milk" in its vault "Copy"
      And the second computer has the same file "Food/Menu.pdf" in its vault "Copy"
      When the person moves the folder "Food" to "Meals" in the vault "Test"
      Then the second computer has the note "Meals/Shopping list" with the text "milk" in its vault "Copy"
      And the second computer has the same file "Meals/Menu.pdf" in its vault "Copy"
      And the second computer does not have the file "Food/Menu.pdf" in its vault "Copy"

    Scenario: A moved folder with many files arrives complete
      Given the person adds 20 files to the folder "Food" in the vault "Test"
      And the second computer has 20 files in its folder "Food" in its vault "Copy"
      When the person moves the folder "Food" to "Meals" in the vault "Test"
      Then the second computer has 20 files in its folder "Meals" in its vault "Copy"

  Rule: A moved note keeps its links

    Scenario: A renamed note changes the links to it on the second computer
      Given the person writes the note "Plan" with the text "See [[Shopping list]]" in the vault "Test"
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      When the person moves the file "Shopping list.md" to "Groceries.md" in the vault "Test"
      Then the second computer has the note "Plan" with the text "See [[Groceries]]" in its vault "Copy"
