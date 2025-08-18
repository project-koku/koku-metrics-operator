<<<<<<<< HEAD:vendor/go.yaml.in/yaml/v2/README.md
# YAML support for the Go language
========
# goyaml.v2

This package provides type and function aliases for the `go.yaml.in/yaml/v2` package (which is compatible with `gopkg.in/yaml.v2`).

## Purpose

The purpose of this package is to:
>>>>>>>> 1a92445b ([COST-6518] generate downstream changes and bundle (#674)):vendor/sigs.k8s.io/yaml/goyaml.v2/README.md

1. Provide a transition path for users migrating from the sigs.k8s.io/yaml package to direct usage of go.yaml.in/yaml/v2
2. Maintain compatibility with existing code while encouraging migration to the upstream package
3. Reduce maintenance overhead by delegating to the upstream implementation

## Usage

Instead of importing this package directly, you should migrate to using `go.yaml.in/yaml/v2` directly:

```go
// Old way
import "sigs.k8s.io/yaml/goyaml.v2"

<<<<<<<< HEAD:vendor/go.yaml.in/yaml/v2/README.md
Installation and usage
----------------------

The import path for the package is *go.yaml.in/yaml/v2*.

To install it, run:

    go get go.yaml.in/yaml/v2

API documentation
-----------------

See: <https://pkg.go.dev/go.yaml.in/yaml/v2>

API stability
-------------

The package API for yaml v2 will remain stable as described in [gopkg.in](https://gopkg.in).


License
-------

The yaml package is licensed under the Apache License 2.0. Please see the LICENSE file for details.


Example
-------

```Go
package main

import (
        "fmt"
        "log"

        "go.yaml.in/yaml/v2"
)

var data = `
a: Easy!
b:
  c: 2
  d: [3, 4]
`

// Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type T struct {
        A string
        B struct {
                RenamedC int   `yaml:"c"`
                D        []int `yaml:",flow"`
        }
}

func main() {
        t := T{}
    
        err := yaml.Unmarshal([]byte(data), &t)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- t:\n%v\n\n", t)
    
        d, err := yaml.Marshal(&t)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- t dump:\n%s\n\n", string(d))
    
        m := make(map[interface{}]interface{})
    
        err = yaml.Unmarshal([]byte(data), &m)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- m:\n%v\n\n", m)
    
        d, err = yaml.Marshal(&m)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- m dump:\n%s\n\n", string(d))
}
========
// Recommended way
import "go.yaml.in/yaml/v2"
>>>>>>>> 1a92445b ([COST-6518] generate downstream changes and bundle (#674)):vendor/sigs.k8s.io/yaml/goyaml.v2/README.md
```

## Available Types and Functions

All public types and functions from `go.yaml.in/yaml/v2` are available through this package:

### Types

- `MapSlice` - Encodes and decodes as a YAML map with preserved key order
- `MapItem` - An item in a MapSlice
- `Unmarshaler` - Interface for custom unmarshaling behavior
- `Marshaler` - Interface for custom marshaling behavior
- `IsZeroer` - Interface to check if an object is zero
- `Decoder` - Reads and decodes YAML values from an input stream
- `Encoder` - Writes YAML values to an output stream
- `TypeError` - Error returned by Unmarshal for decoding issues

### Functions

- `Unmarshal` - Decodes YAML data into a Go value
- `UnmarshalStrict` - Like Unmarshal but errors on unknown fields
- `Marshal` - Serializes a Go value into YAML
- `NewDecoder` - Creates a new Decoder
- `NewEncoder` - Creates a new Encoder
- `FutureLineWrap` - Controls line wrapping behavior

## Migration Guide

To migrate from this package to `go.yaml.in/yaml/v2`:

1. Update your import statements:
   ```go
   // From
   import "sigs.k8s.io/yaml/goyaml.v2"
   
   // To
   import "go.yaml.in/yaml/v2"
   ```

2. No code changes should be necessary as the API is identical

3. Update your go.mod file to include the dependency:
   ```
   require go.yaml.in/yaml/v2 v2.4.2
   ```

## Deprecation Notice

All types and functions in this package are marked as deprecated. You should migrate to using `go.yaml.in/yaml/v2` directly.
