# QuTeDB — The QUBIC Test Database

## Requirements

- [Go](https://golang.org) version 1.10 or newer

Moreover, the following Go libraries should be installed using `go get`:

- [Fitsio](https://github.com/astrogo/fitsio), to read and write FITS files
- [go-chart](https://github.com/wcharczuk/go-chart), to produce charts
- [GORM](https://github.com/jinzhu/gorm), an ORM
- [go-sqlite3](http://mattn.github.io/go-sqlite3), a database driver for [SQLite](https://www.sqlite.org/index.html)
- [go.uuid](https://github.com/satori/go.uuid), for generating random UUIDs
- [Logrus](https://github.com/sirupsen/logrus), for structured logging
- [simple-scrypt](https://github.com/elithrar/simple-scrypt), for encrypting passwords
- [Viper](https://github.com/spf13/viper), for reading configuration files

Note that `go-sqlite3` is a *cgo* package, which means that you must have *gcc*
in your path before downloading and compiling this library. This is usually the
case on UNIX systems; if you are running Windows, you have to install *gcc*
first (e.g., using the [scoop](https://scoop.sh/) package manager).

## Configuration

All the configuration is specified through a file, `config.json`, which is read
from any of the following locations (in this order):

1. The directory from where `qutedb` was run;
2. The directory `.qutedb` within the home folder (i.e., `$HOME` on UNIX
   systems, `%USERPROFILE%` on Windows);
3. The directory `/etc/qutedb` (only on UNIX systems)

Here is an example of configuration file:

`````json
{
    "log_format": "json",
    "port_number": 80
}
`````

The following table contains all the configuration parameters that can be used
in `config.json`. A few of them have sensible defaults, if no value is provided.

| *Parameter*  | *Default* | *Meaning* |
|--------------|-----------|-----------|
| `log_format` | `"text"`    | Format of log messages. Possible values are `"text"` and `"json"` |
| `log_output` | `"-"` | File where to write log messages. If equal to `"-"`, write to stderr; if `"--"`, write to stdout |
| `log_level` | It depends    | Logging level. Possible values are `"error"`, `"warning"`, `"info"`, and `"debug"`, in increasing order of verbosity. The default is `"info"`, unless development mode is turned on |
| `port_number` | `8080`    | Socket port number used for publishing the API and the site |
| `read_timeout` | 15 | Timeout for HTTP read operations, in seconds |
| `static_path` | `static` | Path to the directory containing static files (e.g., images) to serve |
| `server_name` | `127.0.0.1` | Name of the server (e.g., `www.example.com`) |
| `repository_path` | `.` | Path to the folder that contains the QUBIC test data |
| `write_timeout` | 60 | Timeout for HTTP write operations, in seconds |

The following environment variables are recognized and take precedence over the
corresponding keys in `config.json`:

| *Environment variable* | *Key in `config.json`* |
|------------------------|------------------------|
| `QUTEDB_PORT_NUMBER`   | `port_number`          |
| `QUTEDB_SERVER_NAME`   | `server_name`          |
| `QUTEDB_READ_TIMEOUT`  | `read_timeout`         |
| `QUTEDB_WRITE_TIMEOUT` | `write_timeout`        |

## Authentication

The code uses the "scrypt" encryption algorithm to hash users' passwords. See the article [Do not use sha256crypt/sha512crypt](https://pthree.org/2018/05/23/do-not-use-sha256crypt-sha512crypt-theyre-dangerous/) for the reasons behind this choice.

## License

This code is released under the MIT license. See the file LICENSE for more details.
