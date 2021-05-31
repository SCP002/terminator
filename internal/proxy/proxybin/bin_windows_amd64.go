// +build windows
// +build amd64

package proxybin

import _ "embed"

//go:embed proxy_x64.exe
var Bytes []byte
