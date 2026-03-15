# PAT Auth Flow

This document explains how PAT authentication works in `omni-cli` and why it is implemented as a browser login flow instead of a pasted secret.

## Summary

- PAT is not treated as a user-entered API key in this CLI
- PAT setup uses a browser-based OAuth flow
- Org API key setup is still a direct prompt/input flow
- A profile can store:
  - `pat`
  - `org`
  - `both`
- General commands use the profile's `default_auth`
- Org-only commands such as `admin`, `users`, and `scim` always use the org key

## Why

The main Omni API docs distinguish normal API-key usage from MCP OAuth PAT usage. In practice, the PAT flow used by Omni MCP is browser/OAuth based. That means prompting users to paste a PAT during setup is the wrong UX and the wrong model.

## What We Verified

We checked:

- Omni MCP docs
- Omni API authentication docs
- The published `@omni-co/mcp` npm package
- The hosted OAuth discovery metadata on `callbacks.omniapp.co`

Important finding:

- The published `@omni-co/mcp` package does not implement the browser auth flow itself. It is an API-key bridge.
- The usable PAT auth contract comes from Omni's public OAuth discovery metadata, not from the npm package contents.

## OAuth Endpoints Used

These endpoints were publicly discoverable and used to implement PAT login:

- Protected resource metadata:
  - `https://callbacks.omniapp.co/.well-known/oauth-protected-resource`
- Authorization server metadata:
  - `https://callbacks.omniapp.co/.well-known/oauth-authorization-server`
- Dynamic client registration:
  - `https://callbacks.omniapp.co/oauth/register`

Observed metadata includes:

- authorization endpoint:
  - `https://callbacks.omniapp.co/callback/mcp/oauth/authorize`
- token endpoint:
  - `https://callbacks.omniapp.co/callback/mcp/oauth/token`
- scope:
  - `mcp:access`
- PKCE:
  - `S256`
- dynamic client registration:
  - supported
- token endpoint auth method:
  - `none`

## Implemented PAT Flow

The CLI PAT login flow is:

1. Fetch protected resource metadata
2. Fetch authorization server metadata
3. Dynamically register a native OAuth client
4. Start a localhost callback listener on `127.0.0.1`
5. Generate PKCE verifier/challenge and state
6. Open browser to Omni authorization endpoint
7. Receive authorization code on loopback callback
8. Exchange code for bearer token
9. Store the returned token as the PAT credential for the profile

The implementation lives in:

- [internal/cli/pat_login.go](/Users/hawkfry/Documents/open-source/omni-opensource/omni-cli/internal/cli/pat_login.go)

## Public Repo Caveat

This repo is public. The current PAT implementation is considered acceptable to keep in a public repo because:

- it uses public OAuth discovery metadata
- it uses standard OAuth 2.1 native-app patterns
- it does not expose secrets or bypass intended auth behavior

But this should be treated as a currently working public integration point, not as a guaranteed permanent Omni CLI contract.

In other words:

- the flow is public enough to use
- the stability is not guaranteed unless Omni explicitly documents it as a supported standalone CLI auth contract

If Omni changes hosted OAuth endpoints or MCP-related auth behavior, this implementation may need to be updated.

## If This Breaks Later

Re-check:

- Omni MCP docs
- Omni API auth docs
- `callbacks.omniapp.co` OAuth discovery metadata
- whether Omni still supports loopback redirect URIs via dynamic client registration

Do not reintroduce PAT as a pasted token during setup unless Omni explicitly changes their supported auth model.
