# burnside-go

Shared Go module for the Burnside data platform. Imported by both [pg-cdc](https://github.com/burnside-project/pg-cdc) and [pg-warehouse](https://github.com/burnside-project/pg-warehouse).

Pure Go. No CGO.

## Packages

| Package | Purpose |
|---------|---------|
| `manifest` | manifest.json types + read/write |
| `storage` | Storage interface + adapters (filesystem, S3, GCS) |
| `epoch` | Epoch ordering, watermark comparison, filename parsing |
| `types` | Column types, Postgres-to-Parquet mapping, CDC metadata constants |

## Install

```bash
go get github.com/burnside-project/burnside-go@latest
```

## Related Repos

| Repo | Role |
|------|------|
| [pg-cdc](https://github.com/burnside-project/pg-cdc) | CDC server — produces typed Parquet + manifest |
| [pg-warehouse](https://github.com/burnside-project/pg-warehouse) | Analytics client — consumes Parquet via refresh |
| **burnside-go** (this repo) | Shared types — the contract between producer and consumer |

## Versioning

Semver tags: `v0.1.0`, `v0.2.0`, etc. No release candidates — this is a library, not a binary.

Both pg-cdc and pg-warehouse pin the same version in their `go.mod`.

## License

[Apache License 2.0](LICENSE)
