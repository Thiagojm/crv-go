# Third-party notices — CRV Go

This local developer distribution includes the original `farsight/` catalog, the Inter font, and Go dependencies linked into the executable. Each extracted package contains this notice, available dependency license texts under `LICENSES/`, and the Inter OFL license. It is unsigned and has not been published.

## Farsight catalog

The preserved source metadata credits the **Farsight Institute / Courtney Brown and third-party photographers** and states: “Local research archive only.” The catalog also retains source URLs and individual credits when supplied; the application presents them in the authorized post-choice views.

Sources captured on 2026-09-21:

- [Farsight SRV](https://farsight.org/SRV/)
- [Pool A](https://farsight.org/sponsors/PoolA/)
- [Pool B](https://farsight.org/sponsors/PoolB/jumbledpoollistB)
- [Pool C](https://farsight.org/sponsors/PoolC/jumbledpoollistC)

The package preserves `farsight/catalog-unified.json` and `farsight/images/` without rewriting their credits or original usage statement. The source describes local research archive use; this notice grants no additional rights to third-party photographs. Individual records retain their supplied credits and source links.

## Inter font

Inter by Rasmus Andersson / The Inter Project Authors is distributed under the **SIL Open Font License 1.1 (OFL-1.1)**. The full license is included at `LICENSES/Inter-OFL-1.1.txt` and in the repository at `docs/INTER-OFL.txt`.

## Go dependencies

The executable includes code from these dependencies. Available license texts are copied from the resolved module cache into `LICENSES/go/`; `go.sum` records the module content checksums.

| Module | License |
| --- | --- |
| `github.com/go-chi/chi/v5` | MIT |
| `golang.org/x/sys` | BSD-3-Clause |
| `modernc.org/sqlite` | BSD-3-Clause; also includes SQLite components under a public-domain blessing, see `LICENSE-SQLITE` |
| `github.com/dustin/go-humanize` | MIT |
| `github.com/google/uuid` | BSD-3-Clause |
| `github.com/mattn/go-isatty` | MIT |
| `github.com/ncruces/go-strftime` | MIT |
| `github.com/remyoudompheng/bigfft` | BSD-3-Clause |
| `modernc.org/libc` | BSD-3-Clause; also see the module's third-party notice |
| `modernc.org/mathutil` | BSD-3-Clause |
| `modernc.org/memory` | BSD-3-Clause |

The Go standard library license is included at `LICENSES/go/Go-Standard-Library-BSD-3-Clause.txt`.

## npm dependencies

The distributed frontend is built from the versions pinned in `package-lock.json`; Node.js, npm, development tools, and Playwright are not needed to run the application. Each package notice lists the npm lock packages, versions, declared licenses, and integrity hashes. License texts found in the installed packages are copied to `LICENSES/npm/`.

## Package-specific evidence

`THIRD_PARTY_NOTICES.txt` in each extracted archive records the target and actual Go/Node.js/npm versions, SHA-256 digests of `go.mod`, `go.sum`, `package-lock.json`, and the catalog inventory, along with the inventory format, capture date, and declared source-file count. `packages/SHA256SUMS.txt` lists SHA-256 digests for the generated tar.gz archives.

The manifest digests identify the dependency evidence used for each package; individual npm integrity values are in the lockfile and are repeated in the package notice. The binaries are built with `CGO_ENABLED=0` for Windows amd64 and Linux amd64. Cross-compilation does not verify runtime on the target operating system.

## CRV Go project

The CRV Go source code is released under the MIT License (see `LICENSE` in the repository root and `LICENSE.txt` in distributed packages). This notice document specifically covers bundled third-party material, fonts, libraries, and the Farsight catalog archive.
