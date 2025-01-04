//go:build windows && 386

package bin

import _ "embed"

//go:embed proxy_386.exe
var Bytes []byte
