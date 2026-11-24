Feature: Restore an identity from two sheets
  A person gets the identity back from any two sheets of a recovery kit.

  As a person who lost every device
  I want to restore my identity from two sheets of my recovery kit
  So that I do not need the 12 words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person creates an identity
    And the person downloads the recovery kit
    And the person deletes the identity from this device

  Rule: Two sheets restore the same identity

    Scenario: Two sheets give back the same identity
      When the person restores the identity from the sheets "2 and 3"
      Then this device has an identity
      And the identity fingerprint is the same as before

    Scenario: The sheets and the words give the same identity
      Given the person restores the identity from the sheets "1 and 2"
      And the person deletes the identity from this device
      When the person restores the identity from its recovery words
      Then the identity fingerprint is the same as before

  Rule: One sheet restores nothing

    Scenario: One sheet is refused
      When the person restores the identity from the sheets "1"
      Then the restore dialog says "one more sheet needed"
      And this device has no identity

    Scenario: A sheet with a mistake is refused
      When the person restores the identity from the sheets "1 and 3" with a wrong word in the sheet 3
      Then the identity warns about "sheets are not valid"
      And this device has no identity
