# payments-go

Reusable Go helpers for matching Polish bank transfer notifications to orders.

The module provides:

- secure generation and validation of short payment codes;
- amount parsing to grosz without floating-point rounding;
- parsers for Alior and generic Polish bank notification messages;
- transfer-title matching, including common `I`/`1` and `O`/`0` substitutions;
- an IMAP mailbox reader that marks only accepted messages as seen.

Import as `github.com/mleczakm/payments-go`. Application-specific persistence, order workflows, and notifications stay in each consuming application.
