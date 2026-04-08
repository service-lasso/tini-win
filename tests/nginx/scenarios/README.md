# nginx test scenarios

These repo-local nginx configs are used to exercise `tini-win` against controlled lifecycle cases without relying on donor paths.

## Scenarios

### `healthy`
- Starts cleanly
- Exposes `/health` on the chosen port
- Used for normal launch + graceful stop proof

### `no-health`
- Starts cleanly
- Does not expose `/health`
- Used to simulate a healthy process with an application-level probe mismatch

### `invalid-config`
- Intentionally invalid nginx config
- Used to prove fast startup failure / non-zero exit handling

## Rendering

Use `scripts\render-nginx-test-config.ps1` to materialize a scenario into a runnable working directory.

Example:

```powershell
.\scripts\render-nginx-test-config.ps1 -Scenario healthy -Port 18080 -OutputDir .\tmp\nginx-healthy
```
