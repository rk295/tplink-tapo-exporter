/*
This package is a Go client library for the Tapo (https://www.tapo.com/uk/) range of smart plugs.

To print the Nick Name of a specific smart plug do something like this:

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

There is an example [`examples/p110`](examples/p110) which collects information from a P110 smart plug with energy monitoring.
*/
package tapo
