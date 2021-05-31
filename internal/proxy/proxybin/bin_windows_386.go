// +build windows
// +build 386

package proxybin

import _ "embed"

//go:embed proxy_x32.exe
var Bytes []byte
