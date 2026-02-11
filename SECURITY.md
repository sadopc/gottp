# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in gottp, please report it responsibly.

**Do not open a public issue for security vulnerabilities.**

Instead, please email: **albayrak.serdar8@gmail.com** or use [GitHub's private vulnerability reporting](https://github.com/sadopc/gottp/security/advisories/new).

### What to Include

- A description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 1 week
- **Fix release**: Depending on severity, typically within 2 weeks for critical issues

### Scope

The following areas are in scope:

- HTTP request handling and response processing
- Environment variable interpolation and secret handling
- Collection file parsing (YAML)
- Import/export functionality (curl, Postman, Insomnia, OpenAPI, HAR)
- JavaScript scripting engine sandboxing
- Authentication implementations (OAuth2, AWS Signature V4, Digest)
- TLS/mTLS configuration
- Cookie handling
- SQLite history storage
- Mock server

### Out of Scope

- Issues in third-party dependencies (report upstream)
- Denial of service through large input files
- Issues requiring physical access to the machine

## Security Best Practices for Users

- Keep gottp updated to the latest version
- Use environment files for secrets instead of hardcoding in collections
- Enable AES-256-GCM encryption for sensitive environment variables
- Review imported collections before executing requests
- Be cautious with pre/post-request scripts from untrusted sources
