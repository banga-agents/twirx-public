# Public receipt policy

Public receipt copies must be redacted before commit.

Redact at minimum:

- home and billing addresses;
- phone numbers;
- personal email where unnecessary;
- payment card and bank details;
- tax identifiers;
- account recovery data;
- access tokens and service credentials;
- third-party personal information.

Retain the original outside Git where required for accounting. A public expense record may include the SHA-256 digest of the original to demonstrate that the retained record has not changed without publishing it.
