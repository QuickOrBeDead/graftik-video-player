# Release Process

## Commit Messages

No enforced format — push to `main` freely. The CI workflow runs typecheck and `go vet` on every push.

## Version Scheme

Semantic versioning: `major.minor.patch`

- **Stable**: `0.1.0`, `0.2.0`, `1.0.0`, etc.
- **Prerelease (beta)**: `0.1.0-beta.1`, `0.1.0-beta.2`, etc.

## How to Release

1. Push all desired changes to `main`
2. Go to GitHub → **Actions** → **Create Release** → **Run workflow**
3. Fill in:
   - **Version**: e.g. `0.1.0` for stable or `0.1.0-beta.1` for beta
   - **Prerelease**: check if this is a beta
4. Wait ~15 minutes
5. The release appears on the repo's **Releases** page with both `.deb` (Linux) and `.exe` (Windows)

## Beta vs Stable

| Tag | GitHub Release | App auto-updater |
|-----|---------------|------------------|
| `v0.1.0-beta.1` | Prerelease | Only if user opts into betas |
| `v0.1.0` | Stable | All users |

The auto-updater in the app checks `GET /releases/latest`, which returns the latest **non-prerelease** by default. Beta users need a manual setting to receive prereleases.
