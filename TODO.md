# TODO

## Proxy

Backend proxy server to handle OAuth token exchange so users don't need their own WHOOP app credentials.

### Overview

- Host a small Go server (Fly.io free tier)
- Server holds `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET`
- Users run `thoop auth` → opens browser to proxy → proxy handles OAuth → returns token to localhost

### Endpoints

```
GET  /auth/start     → redirect to WHOOP OAuth with client_id, state, redirect to /auth/callback
GET  /auth/callback  → receive code, exchange for token using client_secret, redirect token to user's localhost
```

### Flow

```
1. User runs `thoop auth`
2. TUI starts local server on localhost:8080
3. Opens browser to https://thoop-proxy.fly.dev/auth/start?local_port=8080
4. Proxy generates state, redirects to WHOOP OAuth
5. User logs in, approves
6. WHOOP redirects to https://thoop-proxy.fly.dev/auth/callback?code=xxx&state=yyy
7. Proxy exchanges code for token (using secret stored on server)
8. Proxy redirects to http://localhost:8080/callback?access_token=xxx&refresh_token=yyy&expires_in=zzz
9. Local TUI receives token, saves to ~/.thoop/thoop.db
10. Done
```

### Files to Create

- [ ] `cmd/proxy/main.go` - proxy server entry point
- [ ] `internal/proxy/handlers.go` - /auth/start and /auth/callback handlers
- [ ] `internal/proxy/config.go` - proxy config (reads secrets from env)
- [ ] `fly.toml` - Fly.io deployment config
- [ ] Update `cmd/thoop/auth.go` - point to proxy instead of direct WHOOP OAuth

### Security Considerations

- State parameter validation (CSRF protection)
- Only redirect tokens to localhost (prevent token theft)
- Rate limiting on proxy endpoints
- HTTPS only on proxy

### Deployment

```bash
fly launch
fly secrets set WHOOP_CLIENT_ID=xxx WHOOP_CLIENT_SECRET=yyy
fly deploy
```
