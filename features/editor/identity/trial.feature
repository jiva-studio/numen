@live
Feature: A trial of a plan of Numen Sync

  A person who starts with a trial of a plan sees the day the trial ends and keeps the access until then, also when
  the trial is canceled.

  As a person who uses Numen
  I want to see when my trial ends
  So that I know how long I can sync before I pay

  Background:
    Given a sync server that sells the plans "Basic" with 10 GB and "Plus" with 100 GB runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"

  @own-server
  Scenario: A subscription that starts with a trial shows the day it ends
    When the person starts a trial of the plan "Plus"
    Then the subscription shows the day the trial ends

  @own-server
  Scenario: A trial that is canceled keeps the access until it ends
    Given the person starts a trial of the plan "Plus"
    When the person cancels the trial
    Then the subscription shows the plan "Plus" with 100 GB
