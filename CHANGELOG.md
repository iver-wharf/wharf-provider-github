# Wharf GitHub plugin changelog

This project tries to follow [SemVer 2.0.0](https://semver.org/).

<!--
	When composing new changes to this list, try to follow convention.

	The WIP release shall be updated just before adding the Git tag.
	From (WIP) to (YYYY-MM-DD), ex: (2021-02-09) for 9th of Febuary, 2021

	A good source on conventions can be found here:
	https://changelog.md/
-->

## v2.1.0 (WIP)

- Added support for the TZ environment variable (setting timezones ex.
  `"Europe/Stockholm"`) through the tzdata package. (#25)

- Added config loading from YAML files using
  `github.com/iver-wharf/wharf-core/pkg/config` together with new config models
  for configuring wharf-provider-github. See `config.go` or the reference
  documentation on the `Config` type for information on how to configure
  wharf-provider-github. (#19, #29)

- Added config for setting bind address and port. (#14, #19)

  - Environment variable: `WHARF_HTTP_BINDADDRESS`
  - YAML: `http.bindAddress`

- Added config for loading extra certificates bundle, in addition to the
  system's certificates. (#19)

  - Environment variable: `WHARF_CA_CERTSFILE`
  - YAML: `ca.certsFile`

- Added logging library `github.com/iver-wharf/wharf-core/pkg/logger` instead
  of `fmt.Println` throughout the repository, as well as the Gin integration
  from `github.com/iver-wharf/wharf-core/pkg/ginutil`. (#20)

- Changed version of `github.com/iver-wharf/wharf-core` from pre release to
  v1.3.0. (#19, #28, #30, #43)

- Changed version of `github.com/iver-wharf/wharf-api-client-go`
  from v1.3.0 -> v1.3.1. (#31)

- Changed to return IETF RFC-7807 compatible problem responses on failures
  instead of solely JSON-formatted strings. (#16)

- Added Makefile to simplify building and developing the project locally.
  (#24, #26, #27, #28)

- Added logging and custom exit code when app fails to bind the IP address and
  port. (#28)

- Removed `internal/httputils`, which was moved to
  `github.com/iver-wharf/wharf-core/pkg/cacertutil`. (#30)

- Changed version of Docker base images, relying on "latest" patch version:

  - Alpine: 3.14.0 -> 3.14 (#33)
  - Golang: 1.16.5 -> 1.16 (#33)

- Removed `UploadURL` field from the `importBody` struct, and all references to
  `wharfapi.Provider.UploadURL`, which will be removed in wharf-api v5.0.0 as it
  did not provide any functionality. (#35)

- Changed docker-build scripts for easier windows building. (#44)

## v2.0.0 (2021-07-12)

- BREAKING: Changed Wharf API dependency to v4.1.0. This provider now uses call
  order to getToken -> getProvider -> GitHub Operations instead of the previous
  getProvider -> getToken -> GitHub operations.  (#10)

- Added AvatarURL and GitURL to the `PUT /api/project` body. (#10)

- Changed version of wharf-api-client-go from 1.2.0 -> 1.3.0 (#10)

- Changed version of Go runtime from 1.13 -> 1.16. (#5)

- Changed version of Docker base images:

  - Alpine: 3.13.4 -> 3.14.0 (#10, #17)
  - Golang: 1.13.4 -> 1.16.5 (#5, #17)

- Changed from POST to PATCH calls for tokens & providers to eliminate entry
  duplication. (#10)

- Added environment var `BIND_ADDRESS` for setting bind address and port. (#14)

- Added endpoint `GET /version` that returns an object of version data of the
  API itself. (#5)

- Added Swagger spec metadata such as version that equals the version of the
  API, contact information, and license. (#5)

- Fixed importing from a group different than the authenticated user. (#13)

## v1.1.1 (2021-04-09)

- Added CHANGELOG.md to repository. (!6)

- Added `.dockerignore` to make Docker builds agnostic to wether you've ran
  `swag init` locally. (!8)

- Changed to use new open sourced Wharf API client
  [github.com/iver-wharf/wharf-api-client-go](https://github.com/iver-wharf/wharf-api-client-go)
  and bumped said package version from v1.1.0 to v1.2.0. (!7)

- Changed base Docker image to be `alpine:3.13.4` instead of `scratch` to get
  certificates from the Alpine package manager, APK, instead of embedding a list
  of certificates inside the repository. (#1)

## v1.1.0 (2021-01-07)

- Changed version of Wharf API Go client, from v0.1.5 to v1.1.0, that contained
  a lot of refactors in type and package name changes. (!4, !5)

## v1.0.0 (2020-11-27)

- Removed groups table, a reflection of the changes from the API v1.0.0. (!3)

## v0.8.0 (2020-11-03)

- Added missing `go.sum` file. (!1)

## v0.7.1 (2020-01-22)

- *Version bump.*

## v0.7.0 (2020-01-22)

- *Version bump.*

## v0.6.0 (2020-01-22)

- Added repo, split from mono-repo. (baad06dc)

- Added `.wharf-ci.yml` to build Docker images. (36e299cb)

- Changed Docker build to use Go modules via `GO111MODULE=on` environment
  variable. (34768c1e)

- Fixed reference Go modules in `go.mod`. (daa930a8)

- Fixed Docker build to use `go.mod` instead of explicit references. (97d15055)
