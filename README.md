# GitHub provider for Wharf

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/d2d48df2187247048cdf992e9a5f576d)](https://www.codacy.com/gh/iver-wharf/wharf-provider-github/dashboard?utm_source=github.com\&utm_medium=referral\&utm_content=iver-wharf/wharf-provider-github\&utm_campaign=Badge_Grade)

Import Wharf projects from GitHub repositories. Mainly focused on importing
from github.com, importing from GitHub EE is not well tested.

## Components

- HTTP API using the [gin-gonic/gin](https://github.com/gin-gonic/gin)
  web framework.

- Swagger documentation generated using
  [swaggo/swag](https://github.com/swaggo/swag) and hosted using
  [swaggo/gin-swagger](https://github.com/swaggo/gin-swagger)

- GitHub API client using
  [google/go-github](https://github.com/google/go-github)

## Configuring

The wharf-provider-github program can be configured via environment variables
and through optional config files. See the docs on the `Config` type over at:
<https://pkg.go.dev/github.com/iver-wharf/wharf-provider-github#Config>

## Development

1. Install Go 1.16 or later: <https://golang.org/>

2. Install dependencies using [GNU Make](https://www.gnu.org/software/make/) or 
   [GNUWin32](http://gnuwin32.sourceforge.net/install.html)

   ```console
   $ make deps
   ```

3. Generate the Swagger files (this has to be redone each time the swaggo
   documentation comments has been altered):

   ```console
   $ make swag
   ```

4. Start hacking with your favorite tool. For example VS Code, GoLand,
   Vim, Emacs, or whatnot.

## Releasing

Replace the "v2.0.0" in `make docker version=v2.0.0` with the new version. Full
documentation can be found at [Releasing a new version](https://iver-wharf.github.io/#/development/releasing-a-new-version).

Below are just how to create the Docker images using [GNU Make](https://www.gnu.org/software/make/)
or [GNUWin32](http://gnuwin32.sourceforge.net/install.html):

```console
$ make docker version=v2.0.0
STEP 1: FROM golang:1.16.5 AS build
STEP 2: WORKDIR /src
--> Using cache de3476fd68836750f453d9d4e7b592549fa924c14e68c9b80069881de8aacc9b
--> de3476fd688
STEP 3: ENV GO111MODULE=on
--> Using cache 4f47a95d0642dcaf5525ee1f19113f97911b1254889c5f2ce29eb6f034bd550b
--> 4f47a95d064
STEP 4: RUN go get -u github.com/swaggo/swag/cmd/swag@v1.7.0
...

Push the image by running:
docker push quay.io/iver-wharf/wharf-provider-github:latest
docker push quay.io/iver-wharf/wharf-provider-github:v2.0.0
```

## Linting

```sh
make deps # download linting dependencies

make lint

make lint-go # only lint Go code
make lint-md # only lint Markdown files
```

Some errors can be fixed automatically. Keep in mind that this updates the
files in place.

```sh
make lint-fix

#make lint-fix-go # Go linter does not support fixes
make lint-fix-md # only lint and fix Markdown files
```

---

Maintained by [Iver](https://www.iver.com/en).
Licensed under the [MIT license](./LICENSE).
