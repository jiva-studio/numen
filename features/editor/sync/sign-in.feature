Feature: Numen Sync signs in

  The plugin "Numen Sync" signs in to the sync server with the identity of this device.

  As a person who uses Numen on more than one device
  I want Numen Sync to sign in with my identity
  So that the server knows my device without a password

  Background:
    Given the empty vault "Test"
    And a sync server runs

  Rule: Numen Sync signs in with the identity of this device

    Scenario: Numen Sync signs in to the server
      Given the person creates an identity
      When the application "editor" starts
      Then the sync server has a session for this device

    Scenario: The server knows the identity
      Given the person creates an identity
      When the application "editor" starts
      Then the sync server has a session for the fingerprint of the identity

    Scenario: Numen Sync does not sign in without an identity
      When the application "editor" starts
      Then the plugin "Numen Sync" warns about "No identity on this device"
      And the sync server has no session

    Scenario: Numen Sync signs in after the identity is created
      Given the application "editor" starts
      When the person creates an identity
      Then the sync server has a session for this device

  Rule: Numen Sync asks for an update when the server needs a newer protocol

    Scenario: An old Numen is asked to update
      Given the person creates an identity
      And the sync server needs a newer protocol
      When the application "editor" starts
      Then the plugin "Numen Sync" warns about "Update Numen"
      And the sync server has no session

  Rule: A session ends when the server ends it

    Scenario: Numen Sync signs in again after the server ends the session
      Given the person creates an identity
      And the application "editor" starts
      When the sync server ends the session of this device
      Then the sync server has a session for this device
