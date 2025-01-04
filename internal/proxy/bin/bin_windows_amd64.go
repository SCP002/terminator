//go:build windows && amd64

package bin

import _ "embed"

//go:embed proxy_amd64.exe
var Bytes []byte
