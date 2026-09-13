# OAuth authorization on a remote server

Clocky uses a provider-neutral configuration server from `internal/configweb`
and integration contracts from `internal/oauth`. The server exposes a persistent
settings dashboard, starts provider authorization, receives callbacks, and lets
the user disconnect or replace an account.

The listener binds only to a loopback address for local use.

## Spotify setup

Register this exact redirect URI in the Spotify developer dashboard:

```text
http://127.0.0.1:8888/callback
```

Configure the server-side `.env` file:

```text
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_REDIRECT_URI=http://127.0.0.1:8888/callback
```

Clocky uses Authorization Code with PKCE, so a client secret is not required.

## Authorize locally

1. Start Clocky.
2. Open `http://127.0.0.1:8888/` in the browser.
3. Select **Connect account** and complete consent.

The same page can later be used to switch to a different account or disconnect
the current account. **Switch account** opens Spotify's consent dialog again;
the account selected in the browser becomes the account used by Clocky.

The Spotify refresh token is stored in the user's configuration directory, normally
`~/.config/clocky/spotify-token.json`, with file mode `0600`.

If the callback port changes, update the provider dashboard and the redirect URI.
All redirect URI components must match exactly.

## Reusing the callback server

Other OAuth providers can implement the `oauth.Integration` interface and be
registered with `configweb.Server`. The configuration server owns HTTP routing,
templates, CSRF protection, callback-state validation, and account actions.
Provider endpoints, scopes, token exchange, credential storage, and domain
mapping remain in the relevant `internal/data` provider.
