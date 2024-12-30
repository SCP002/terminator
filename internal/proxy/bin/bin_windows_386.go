//go:build windows && 386

package bin

import _ "embed"

//go:embed proxy_x32.exe
var Bytes []byte
