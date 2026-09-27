# gopipe

gopipe is a Windows-native CLI pipeline wrapper for legacy RPA and ETL tools used in
telecom operations.

## Status

| Subcommand | Phase | State |
|---|---|---|
| `pipeline` (default) | 2 | implemented |
| `db extract` | 3 | implemented |
| `db load` | 4 | stub |
| `run` | 5 | stub |
| `registry` | 5 | stub |
| `shell` | 2 | stub |
| `gui` | 6 | stub |
| `version`, `help` | 1 | implemented |

## Build

```bat
build_all.bat
```

Produces `gopipe.exe` and `gopipe-gui.exe` for `windows/amd64` with `CGO_ENABLED=0`,
injecting version, commit and build time into `gopipe/internal/version.*` via
`-ldflags`. Direct builds also work: `CGO_ENABLED=0 go build ./...`.

## Extract a database query

Create `task.json` (use a read-only DB account for extracts):

```json
{
  "driver": "postgres",
  "dsn": "postgres://user:password@host:5432/database?sslmode=require",
  "sql": "select id, name from customers",
  "output": "customers.csv",
  "format": "csv",
  "batch_size": 1000
}
```

```sh
gopipe db extract --json task.json
gopipe db extract --yaml task.yaml   # the same keys in YAML
# gopipe db --json task.json         # shorthand for db extract
```

Supported drivers: `sybase` (`tds`), `oracle` (`go-ora/v2`), `mssql`
(`go-mssqldb`), `postgres` (`pgx/v5`), and `gaussdb` (PostgreSQL protocol).
Output formats: `csv` (default), `tsv`, `json` (array of row objects), `jsonl`
(one row object per line). Rows stream to a temporary file in the output
directory and replace the destination only after success. A batch file may
contain a `tasks` list of the same objects, executed in order. `batch_size` is
validated for task-file compatibility; extraction currently streams one row at a
time regardless of this value.

SQL supports environment-variable templates such as `{{.REPORT_DATE}}`. Missing
variables are errors. These templates are **text substitution, not SQL bind
parameters**: never place untrusted input into them. Protect task files and
DSNs like credentials; do not commit production DSNs to Git. Output paths are
relative to the current working directory unless absolute.

## Tests

```sh
CGO_ENABLED=0 go build ./...
go test ./...
```

The database-extract tests use an in-process `database/sql` fixture and require
no real database. Live DB connectivity should also be verified against each
deployment's server and DSN configuration.
