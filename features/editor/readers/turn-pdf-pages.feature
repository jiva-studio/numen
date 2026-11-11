Feature: Turn PDF pages
  A person goes to a page of a PDF by its number or with the buttons.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | sample    |
      | assets/pages.pdf | pages.pdf |
    And the application "editor" starts
    And the person opens the file at "assets/pages.pdf"

  Scenario: A page number is typed
    When the person goes to page "12" in the PDF tab at "assets/pages.pdf"
    Then the PDF tab at "assets/pages.pdf" is at page "12"

  Scenario: A typed page stays after the application starts again
    Given the person goes to page "12" in the PDF tab at "assets/pages.pdf"
    When the application "editor" restarts
    Then the PDF tab at "assets/pages.pdf" is at page "12"

  Scenario: The next page button turns the page
    When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    And the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    And the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    Then the PDF tab at "assets/pages.pdf" is at page "4"

  Scenario: The buttons stop at the first page
    Then the button "Next page" in the PDF tab at "assets/pages.pdf" is enabled
    And the button "Previous page" in the PDF tab at "assets/pages.pdf" is disabled

  Scenario: The previous page button works after a turn
    Given the button "Previous page" in the PDF tab at "assets/pages.pdf" is disabled
    When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    Then the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled

  Scenario: The buttons stop at the last page
    When the person goes to page "40" in the PDF tab at "assets/pages.pdf"
    Then the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled
    And the button "Next page" in the PDF tab at "assets/pages.pdf" is disabled
