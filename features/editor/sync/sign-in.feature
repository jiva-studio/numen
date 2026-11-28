Feature: Numen Sync signs in

  Numen Sync signs in to the sync server with the identity of this device.

  As a person who uses Numen on more than one device
  I want Numen Sync to sign in with my identity
  So that the server knows my device without a password

  Background:
    Given the empty vault "Test"
    And a sync server runs

  Rule: Numen Sync signs in with the identity of this device

    @live
    Scenario: Numen Sync signs in to the server
      Given the person creates an identity
      When the application "editor" starts
      Then this device is signed in to the sync server

    @live
    Scenario: Numen Sync does not sign in without an identity
      When the application "editor" starts
      Then the person is told "No identity on this device"
      And this device is not signed in to the sync server

    @live
    Scenario: Numen Sync signs in after the identity is created
      Given the application "editor" starts
      When the person creates an identity
      Then this device is signed in to the sync server

  Rule: Numen Sync signs in again when the server signs it out

    @stand
    Scenario: Numen Sync signs in again after the server signs this device out
      Given the person creates an identity
      And the application "editor" starts
      When the sync server signs this device out
      Then this device is signed in to the sync server
