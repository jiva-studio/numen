Feature: Restore an identity from two sheets
  A person gets the account back from any two sheets of a recovery kit.

  As a person who lost every device
  I want to restore my account from two sheets of my recovery kit
  So that I do not need the 12 words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person creates an identity
    And the person downloads the recovery kit
    And the person deletes the identity from this device

  Rule: Two sheets restore the same account

    Scenario: Two sheets give back the same account
      When the person restores the identity from the sheets "2 and 3"
      Then this device has an identity
      And the account fingerprint is the same as before

    Scenario: The sheets and the words give the same account
      Given the person restores the identity from the sheets "1 and 2"
      And the person deletes the identity from this device
      When the person restores the identity from its recovery words
      Then the account fingerprint is the same as before

  Rule: One sheet restores nothing

    Scenario: One sheet is refused
      When the person restores the identity from the sheets "1"
      Then the identity warns about "one more sheet"
      And this device has no identity

    Scenario: A sheet with a mistake is refused
      When the person restores the identity from the sheets "1 and 3" with a wrong word in the sheet 3
      Then the identity warns about "sheets are not valid"
      And this device has no identity
