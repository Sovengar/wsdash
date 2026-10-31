Feature: wsdash dashboard
  Scenario: Launch configured project
    Given config with project prdash
    When user presses enter
    Then fake launcher records one launch
  Scenario: Toggle scanned projects
    Given scanned projects present
    When user presses tab
    Then hidden list shows
