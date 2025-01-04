//go:build windows && arm64

package bin

import _ "embed"

//go:embed proxy_arm64.exe
var Bytes []byte
