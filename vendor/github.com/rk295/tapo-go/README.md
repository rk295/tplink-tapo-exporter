# Tapo Golang

[![Go Reference](https://pkg.go.dev/badge/github.com/rk295/tapo-go)](https://pkg.go.dev/github.com/rk295/tapo-go) [![Go Report Card](https://goreportcard.com/badge/github.com/rk295/tapo-go)](https://goreportcard.com/report/github.com/rk295/tapo-go) [![GH Action Badge](https://github.com/rk295/tapo-go/actions/workflows/actions.yml/badge.svg?branch=rk-master)](https://github.com/rk295/tapo-go/actions)

This package is a Go client library for the [Tapo](https://www.tapo.com/uk/) range of smart plugs.

## Example Usage

To print the Nick Name of a specific smart plug do something like this:

```go
package main

import (
  "fmt"

  "github.com/rk295/tapo-go"
)

func main() {
  device, err := tapo.NewFromEnv()
  if err != nil {
    panic(err)
  }

  if err := device.Login(); err != nil {
    panic(err)
  }

  deviceInfo, err := device.GetDeviceInfo()
  if err != nil {
    panic(err)
  }

  fmt.Println(deviceInfo.Nickname)
}
```

## Examples

There is an example [ `examples/p110` ](examples/p110) which collects information from a P110 smart plug with energy monitoring.
