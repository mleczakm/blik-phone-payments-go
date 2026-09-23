# BLIK phone payments for Go

Go helpers for the zero-fee, phone-to-phone BLIK instant-payment flow: issue a
short payment title code and reconcile incoming bank notifications to orders.
The module does not integrate a paid payment gateway or charge transaction fees.

The module provides:

- secure generation and validation of short payment codes;
- amount parsing to grosz without floating-point rounding;
- parsers for Alior and generic Polish bank notification messages;
- transfer-title matching, including common `I`/`1` and `O`/`0` substitutions;
- an IMAP mailbox reader that marks only accepted BLIK payment notifications as seen.

Import as `github.com/mleczakm/blik-phone-payments-go`. Application-specific
persistence, order workflows, and notifications stay in each consuming application.
