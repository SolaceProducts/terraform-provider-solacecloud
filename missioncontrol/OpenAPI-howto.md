# Overview

* Retrieve the latest OpenAPI spec from the Solace Cloud API [Solace Cloud OpenAPI Specifications](https://api.solace.dev/cloud/page/openapi-specifications)
* Save as file the `Mission Control - v2.0 - JSON` link - Overwrite [MissionControl-specs.json](./MissionControl-specs.json)

# Install instructions

It is important to ensure that you have your "GOBIN" directory in your path, as this install instruction will place
the code generator API there.

```
export PATH=${PATH}:$GOPATH/bin
```

Then, install the OpenAPI Generator CLI tool if you haven't already:
See [OAPI Codegen Github](https://github.com/oapi-codegen/oapi-codegen)

```shell
# for the binary install
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

You should add the PATH above to your shell profile (e.g., `.bashrc`, `.zshrc`, etc.) to ensure that the `oapi-codegen`
command is available in your terminal next time you open it.

# Usage instructions

Then, generate an updated `missioncontrol.go` Go client library using the OpenAPI Generator:
```shell
oapi-codegen -generate types,client -package missioncontrol MissionControl-specs.json > missioncontrol.go
```

run `go mod tidy` if new packages are introduced.