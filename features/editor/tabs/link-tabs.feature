Feature: Link tabs
  A note opened in a tab opens in the tab that is linked to it.

  As a person who works with several tabs
  I want to link a tab with another tab
  So that the tab I look at follows the note I open

  Scenario: A note is opened in the tab it is linked to
    Given the empty vault "Test"
    And the vault has the following:
      | path                 | text                                        |
      | notes/Weir notes.md  | Water over the weir.                        |
      | notes/Flume notes.md | The flume carries the water onto the wheel. |
    And the application "editor" starts
    And the person opens the note titled "Weir notes"
    And the person runs the command "Link with tab" on the tab titled "Files" with:
      | Search | Weir notes |
    When the person opens the file at "notes/Flume notes.md"
    Then the note tab titled "Flume notes" has "The flume carries the water onto the wheel."
    And the tab titled "Weir notes" is not open
