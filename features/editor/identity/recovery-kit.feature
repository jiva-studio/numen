Feature: Download a recovery kit
  A person saves a recovery kit as a PDF of three sheets. Any two sheets restore the identity; one
  sheet alone reveals nothing.

  As a person who uses Numen on more than one device
  I want a kit of three sheets that I can print and keep in different places
  So that I get my identity back when I lose all my devices and forget the words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person creates an identity

  Rule: The kit is a PDF of three sheets

    Scenario: The kit of a new identity downloads at once
      When the person downloads the recovery kit
      Then the recovery kit is saved as a PDF of 3 sheets
      And each sheet shows the identity fingerprint

    Scenario: A later kit asks for the recovery words
      Given the person downloads the recovery kit
      When the person downloads the recovery kit again with the recovery words
      Then the recovery kit is saved as a PDF of 3 sheets

    Scenario: Words of another identity make no kit
      Given the person downloads the recovery kit
      When the person downloads the recovery kit of another identity
      Then the identity warns about "another identity"
