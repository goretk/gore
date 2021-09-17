// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyPackage(t *testing.T) {

	tests := []struct {
		pkgsName string
		pkgPath  string
		pkgClass PackageClass
	}{
		{"attack", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/attack", ClassMain},
		{"bufio", "C:/Go/src/bufio", ClassSTD},
		{"bytes", "C:/Go/src/runtime", ClassSTD},
		{"", "C:/Go/src/runtime", ClassSTD},
		{"compress/bzip2", "C:/Go/src/compress/bzip2", ClassSTD},
		{"compress/flate", "C:/Go/src/compress/flate", ClassSTD},
		{"compress/gzip", "C:/Go/src/compress/gzip", ClassSTD},
		{"crypto/aes", "C:/Go/src/crypto/aes", ClassSTD},
		{"crypto", "C:/Go/src/crypto", ClassSTD},
		{"crypto/cipher", "C:/Go/src/crypto/cipher", ClassSTD},
		{"crypto/des", "C:/Go/src/crypto/des", ClassSTD},
		{"crypto/dsa", "C:/Go/src/crypto/dsa", ClassSTD},
		{"crypto/ecdsa", "C:/Go/src/crypto/ecdsa", ClassSTD},
		{"crypto/elliptic", "C:/Go/src/crypto/elliptic", ClassSTD},
		{"crypto/hmac", "C:/Go/src/crypto/hmac", ClassSTD},
		{"crypto/md5", "C:/Go/src/crypto/md5", ClassSTD},
		{"crypto/rand", "C:/Go/src/crypto/rand", ClassSTD},
		{"crypto/rc4", "C:/Go/src/crypto/rc4", ClassSTD},
		{"crypto/rsa", "C:/Go/src/crypto/rsa", ClassSTD},
		{"crypto/sha1", "C:/Go/src/crypto/sha1", ClassSTD},
		{"crypto/sha256", "C:/Go/src/crypto/sha256", ClassSTD},
		{"crypto/sha512", "C:/Go/src/crypto/sha512", ClassSTD},
		{"crypto/subtle", "C:/Go/src/crypto/subtle", ClassSTD},
		{"crypto/tls", "C:/Go/src/crypto/tls", ClassSTD},
		{"crypto/tls.(*Config).(crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"crypto/x509", "C:/Go/src/crypto/x509", ClassSTD},
		{"crypto/x509/pkix", "C:/Go/src/crypto/x509/pkix", ClassSTD},
		{"encoding/asn1", "C:/Go/src/encoding/asn1", ClassSTD},
		{"encoding/base64", "C:/Go/src/encoding/base64", ClassSTD},
		{"encoding/binary", "C:/Go/src/encoding/binary", ClassSTD},
		{"encoding/hex", "C:/Go/src/encoding/hex", ClassSTD},
		{"encoding/json.(*arrayEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json", "C:/Go/src/encoding/json", ClassSTD},
		{"encoding/json.(*condAddrEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json.(floatEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json.(*mapEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json.(*ptrEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json.(*sliceEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/json.(*structEncoder).(encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"encoding/pem", "C:/Go/src/encoding/pem", ClassSTD},
		{"errors", "C:/Go/src/errors", ClassSTD},
		{"fmt", "C:/Go/src/fmt", ClassSTD},
		{"github.com/garyburd/redigo/internal", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/garyburd/redigo/internal", ClassVendor},
		{"github.com/garyburd/redigo/redis", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/garyburd/redigo/redis", ClassVendor},
		{"github.com/inconshreveable/go-update", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/inconshreveable/go-update", ClassVendor},
		{"github.com/inconshreveable/go-update/internal/binarydist", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/inconshreveable/go-update/internal/binarydist", ClassVendor},
		{"github.com/inconshreveable/go-update/internal/osext", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/inconshreveable/go-update/internal/osext", ClassVendor},
		{"github.com/moul/http2curl", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/moul/http2curl", ClassVendor},
		{"github.com/parnurzeal/gorequest", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/parnurzeal/gorequest", ClassVendor},
		{"github.com/shirou/gopsutil/cpu", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/cpu", ClassVendor},
		{"github.com/shirou/gopsutil/host", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/host", ClassVendor},
		{"github.com/shirou/gopsutil/internal/common", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/internal/common", ClassVendor},
		{"github.com/shirou/gopsutil/mem", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/mem", ClassVendor},
		{"github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassVendor},
		{"github.com/shirou/gopsutil/process", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/process", ClassVendor},
		{"github.com/takama/daemon", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/takama/daemon", ClassVendor},
		{"go", ".", ClassSTD},
		{"golang.org/x/crypto/curve25519", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/curve25519", ClassVendor},
		{"golang.org/x/crypto/ed25519", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ed25519", ClassVendor},
		{"golang.org/x/crypto/ed25519/internal/edwards25519", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ed25519/internal/edwards25519", ClassVendor},
		{"golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassVendor},
		{"golang.org/x/net/publicsuffix", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/net/publicsuffix", ClassVendor},
		{"gopkg.in/vmihailenco/msgpack%2ev2", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/gopkg.in/vmihailenco/msgpack.v2", ClassVendor},
		{"go.(*struct { net/http", ".", ClassGenerated},
		{"go.struct { net/http", ".", ClassGenerated},
		{"go.(*struct { sync.Mutex; table [64]math/big", ".", ClassGenerated},
		{"go.(*struct { sync.RWMutex; m map[reflect.Type][]encoding/json", ".", ClassGenerated},
		{"go.(*struct { sync.RWMutex; m map[reflect.Type]encoding/json", ".", ClassGenerated},
		{"hash", "C:/Go/src/hash", ClassSTD},
		{"hash/crc32", "C:/Go/src/hash/crc32", ClassSTD},
		{"internal/golang.org/x/net/http2/hpack", "C:/Go/src/internal/golang.org/x/net/http2/hpack", ClassSTD},
		{"internal/singleflight", "C:/Go/src/internal/singleflight", ClassSTD},
		{"internal/syscall/unix", "C:/Go/src/internal/syscall/unix", ClassSTD},
		{"io", "C:/Go/src/io", ClassSTD},
		{"io/ioutil", "C:/Go/src/io/ioutil", ClassSTD},
		{"log", "C:/Go/src/log", ClassSTD},
		{"main", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/lady", ClassMain},
		{"math/big", "C:/Go/src/math/big", ClassSTD},
		{"math", "C:/Go/src/math", ClassSTD},
		{"math/rand", "C:/Go/src/math/rand", ClassSTD},
		{"mime", "C:/Go/src/mime", ClassSTD},
		{"mime/multipart", "C:/Go/src/mime/multipart", ClassSTD},
		{"mime/quotedprintable", "C:/Go/src/mime/quotedprintable", ClassSTD},
		{"minerd", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/minerd", ClassMain},
		{"net", "C:/Go/src/runtime", ClassSTD},
		{"net/http", "C:/Go/src/net/http", ClassSTD},
		{"net/http/cookiejar", "C:/Go/src/net/http/cookiejar", ClassSTD},
		{"net/http.(*envOnce).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2clientConnReadLoop).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2clientStream).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2responseWriterState).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2serverConn).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2stream).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*http2Transport).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http/httputil", "C:/Go/src/net/http/httputil", ClassSTD},
		{"net/http/internal", "C:/Go/src/net/http/internal", ClassSTD},
		{"net/http.(*persistConn).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*response).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*Server).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/http.(*Transport).(net/http", "C:/Go/src/net/http", ClassGenerated},
		{"net/textproto", "C:/Go/src/net/textproto", ClassSTD},
		{"net/url", "C:/Go/src/net/url", ClassSTD},
		{"os", "C:/Go/src/runtime", ClassSTD},
		{"os/exec", "C:/Go/src/os/exec", ClassSTD},
		{"os/exec.(*closeOnce).(os/exec", "C:/Go/src/os/exec", ClassGenerated},
		{"os/user", "C:/Go/src/os/user", ClassSTD},
		{"path", "C:/Go/src/path", ClassSTD},
		{"path/filepath", "C:/Go/src/path/filepath", ClassSTD},
		{"redis", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/redis", ClassMain},
		{"reflect", "C:/Go/src/runtime", ClassSTD},
		{"regexp", "C:/Go/src/regexp", ClassSTD},
		{"regexp.(*onePassInst).regexp/syntax", ".", ClassGenerated},
		{"regexp/syntax", "C:/Go/src/regexp/syntax", ClassSTD},
		{"runtime", "C:/Go/src/runtime", ClassSTD},
		{"runtime/debug", "C:/Go/src/runtime", ClassSTD},
		{"runtime/internal/atomic", "C:/Go/src/runtime/internal/atomic", ClassSTD},
		{"sort", "C:/Go/src/sort", ClassSTD},
		{"st", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/st", ClassMain},
		{"strconv", "C:/Go/src/strconv", ClassSTD},
		{"strings", "C:/Go/src/runtime", ClassSTD},
		{"sync/atomic", "C:/Go/src/runtime", ClassSTD},
		{"sync", "C:/Go/src/runtime", ClassSTD},
		{"syscall", "C:/Go/src/runtime", ClassSTD},
		{"text/template", "C:/Go/src/text/template", ClassSTD},
		{"text/template/parse", "C:/Go/src/text/template/parse", ClassSTD},
		{"text/template.(*Template).text/template/parse", ".", ClassGenerated},
		{"text/template.Template.text/template/parse", ".", ClassGenerated},
		{"time", "C:/Go/src/runtime", ClassSTD},
		{"type", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/lady", ClassGenerated},
		{"type..eq.[0]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.[10]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[10]struct { a string; b text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..eq.[11]struct { a golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.[14]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[1]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.[1]golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/redis", ClassGenerated},
		{"type..eq.[1]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[1]text/template", "C:/Go/src/text/template", ClassGenerated},
		{"type..eq.[1]text/template/parse", "C:/Go/src/text/template", ClassGenerated},
		{"type..eq.[23]struct { a crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..eq.[2]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.[2]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[2]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[38]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[3]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[3]text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..eq.[4]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.[4]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[5]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.[5]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[61]internal/golang.org/x/net/http2/hpack", "C:/Go/src/internal/golang.org/x/net/http2/hpack", ClassGenerated},
		{"type..eq.[6]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.[8]golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.[8]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.compress/flate", "C:/Go/src/compress/flate", ClassGenerated},
		{"type..eq.crypto/cipher", "C:/Go/src/crypto/cipher", ClassGenerated},
		{"type..eq.crypto/ecdsa", "C:/Go/src/crypto/ecdsa", ClassGenerated},
		{"type..eq.crypto/elliptic", "C:/Go/src/crypto/elliptic", ClassGenerated},
		{"type..eq.crypto/rand", "C:/Go/src/crypto/rand", ClassGenerated},
		{"type..eq.crypto/rc4", "C:/Go/src/crypto/rc4", ClassGenerated},
		{"type..eq.crypto/sha1", "C:/Go/src/crypto/sha1", ClassGenerated},
		{"type..eq.crypto/sha256", "C:/Go/src/crypto/sha256", ClassGenerated},
		{"type..eq.crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..eq.crypto/x509", "C:/Go/src/crypto/x509", ClassGenerated},
		{"type..eq.encoding/asn1", "C:/Go/src/encoding/asn1", ClassGenerated},
		{"type..eq.encoding/base64", "C:/Go/src/encoding/base64", ClassGenerated},
		{"type..eq.encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"type..eq.github.com/garyburd/redigo/redis", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/garyburd/redigo/redis", ClassGenerated},
		{"type..eq.github.com/inconshreveable/go-update", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/inconshreveable/go-update", ClassGenerated},
		{"type..eq.github.com/shirou/gopsutil/cpu", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/cpu", ClassGenerated},
		{"type..eq.github.com/shirou/gopsutil/host", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/host", ClassGenerated},
		{"type..eq.github.com/shirou/gopsutil/mem", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/mem", ClassGenerated},
		{"type..eq.github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..eq.github.com/shirou/gopsutil/process", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/process", ClassGenerated},
		{"type..eq.golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.hash/crc32", "C:/Go/src/hash/crc32", ClassGenerated},
		{"type..eq.internal/golang.org/x/net/http2/hpack", "C:/Go/src/internal/golang.org/x/net/http2/hpack", ClassGenerated},
		{"type..eq.internal/singleflight", "C:/Go/src/internal/singleflight", ClassGenerated},
		{"type..eq.math/rand", "C:/Go/src/math/rand", ClassGenerated},
		{"type..eq.mime/multipart", "C:/Go/src/mime/multipart", ClassGenerated},
		{"type..eq.net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.net/http/cookiejar", "C:/Go/src/net/http/cookiejar", ClassGenerated},
		{"type..eq.net/http/internal", "C:/Go/src/net/http/internal", ClassGenerated},
		{"type..eq.net/textproto", "C:/Go/src/net/textproto", ClassGenerated},
		{"type..eq.net/url", "C:/Go/src/net/url", ClassGenerated},
		{"type..eq.os/exec", "C:/Go/src/os/exec", ClassGenerated},
		{"type..eq.os/user", "C:/Go/src/os/user", ClassGenerated},
		{"type..eq.regexp/syntax", "C:/Go/src/regexp/syntax", ClassGenerated},
		{"type..eq.struct { a crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..eq.struct { a golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.struct { a string; b text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..eq.struct { F uintptr; cancel chan struct {}; rt net/http.RoundTripper; req *net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.struct { F uintptr; R net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.struct { F uintptr; s *golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.struct { Name string; E *math/big.Int; N *math/big", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.struct { Name string; P *math/big.Int; Q *math/big.Int; G *math/big.Int; Y *math/big", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..eq.struct { net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..eq.text/template", "C:/Go/src/text/template", ClassGenerated},
		{"type..eq.text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..hash.[0]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.[10]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[10]struct { a string; b text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..hash.[11]struct { a golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.[14]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[1]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.[1]golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/src/redis", ClassGenerated},
		{"type..hash.[1]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[1]text/template", "C:/Go/src/text/template", ClassGenerated},
		{"type..hash.[1]text/template/parse", "C:/Go/src/text/template", ClassGenerated},
		{"type..hash.[23]struct { a crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..hash.[2]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.[2]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[2]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[38]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[3]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[3]text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..hash.[4]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.[4]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[5]github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.[5]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[61]internal/golang.org/x/net/http2/hpack", "C:/Go/src/internal/golang.org/x/net/http2/hpack", ClassGenerated},
		{"type..hash.[6]struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.[8]golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.[8]net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.compress/flate", "C:/Go/src/compress/flate", ClassGenerated},
		{"type..hash.crypto/cipher", "C:/Go/src/crypto/cipher", ClassGenerated},
		{"type..hash.crypto/ecdsa", "C:/Go/src/crypto/ecdsa", ClassGenerated},
		{"type..hash.crypto/elliptic", "C:/Go/src/crypto/elliptic", ClassGenerated},
		{"type..hash.crypto/rand", "C:/Go/src/crypto/rand", ClassGenerated},
		{"type..hash.crypto/rc4", "C:/Go/src/crypto/rc4", ClassGenerated},
		{"type..hash.crypto/sha1", "C:/Go/src/crypto/sha1", ClassGenerated},
		{"type..hash.crypto/sha256", "C:/Go/src/crypto/sha256", ClassGenerated},
		{"type..hash.crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..hash.crypto/x509", "C:/Go/src/crypto/x509", ClassGenerated},
		{"type..hash.encoding/asn1", "C:/Go/src/encoding/asn1", ClassGenerated},
		{"type..hash.encoding/base64", "C:/Go/src/encoding/base64", ClassGenerated},
		{"type..hash.encoding/json", "C:/Go/src/encoding/json", ClassGenerated},
		{"type..hash.github.com/garyburd/redigo/redis", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/garyburd/redigo/redis", ClassGenerated},
		{"type..hash.github.com/inconshreveable/go-update", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/inconshreveable/go-update", ClassGenerated},
		{"type..hash.github.com/shirou/gopsutil/cpu", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/cpu", ClassGenerated},
		{"type..hash.github.com/shirou/gopsutil/host", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/host", ClassGenerated},
		{"type..hash.github.com/shirou/gopsutil/mem", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/mem", ClassGenerated},
		{"type..hash.github.com/shirou/gopsutil/net", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/net", ClassGenerated},
		{"type..hash.github.com/shirou/gopsutil/process", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/github.com/shirou/gopsutil/process", ClassGenerated},
		{"type..hash.golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.hash/crc32", "C:/Go/src/hash/crc32", ClassGenerated},
		{"type..hash.internal/golang.org/x/net/http2/hpack", "C:/Go/src/internal/golang.org/x/net/http2/hpack", ClassGenerated},
		{"type..hash.internal/singleflight", "C:/Go/src/internal/singleflight", ClassGenerated},
		{"type..hash.math/rand", "C:/Go/src/math/rand", ClassGenerated},
		{"type..hash.mime/multipart", "C:/Go/src/mime/multipart", ClassGenerated},
		{"type..hash.net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.net/http/cookiejar", "C:/Go/src/net/http/cookiejar", ClassGenerated},
		{"type..hash.net/http/internal", "C:/Go/src/net/http/internal", ClassGenerated},
		{"type..hash.net/textproto", "C:/Go/src/net/textproto", ClassGenerated},
		{"type..hash.net/url", "C:/Go/src/net/url", ClassGenerated},
		{"type..hash.os/exec", "C:/Go/src/os/exec", ClassGenerated},
		{"type..hash.os/user", "C:/Go/src/os/user", ClassGenerated},
		{"type..hash.regexp/syntax", "C:/Go/src/regexp/syntax", ClassGenerated},
		{"type..hash.struct { a crypto/tls", "C:/Go/src/crypto/tls", ClassGenerated},
		{"type..hash.struct { a golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.struct { a net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.struct { a string; b text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"type..hash.struct { F uintptr; cancel chan struct {}; rt net/http.RoundTripper; req *net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.struct { F uintptr; R net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.struct { F uintptr; s *golang.org/x/crypto/ssh", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.struct { Name string; E *math/big.Int; N *math/big", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.struct { Name string; P *math/big.Int; Q *math/big.Int; G *math/big.Int; Y *math/big", "C:/Users/h/CloudStation/Projects/0/ly/lady/vendor/src/golang.org/x/crypto/ssh", ClassGenerated},
		{"type..hash.struct { net/http", "C:/Go/src/net/http", ClassGenerated},
		{"type..hash.text/template", "C:/Go/src/text/template", ClassGenerated},
		{"type..hash.text/template/parse", "C:/Go/src/text/template/parse", ClassGenerated},
		{"unicode", "C:/Go/src/unicode", ClassSTD},
		{"unicode/utf8", "C:/Go/src/unicode/utf8", ClassSTD},
		{"vendor/golang_org/x/net/http2/hpack", "c:/go/src/vendor/golang_org/x/net/http2/hpack", ClassVendor},
		{"vendor/golang_org/x/text/unicode/norm", "c:/go/src/vendor/golang_org/x/text/unicode/norm", ClassVendor},
		{"vendor/golang_org/x/net/dns/dnsmessage", ".", ClassVendor},
		{"vendor/golang_org/x/text/unicode/bidi", "c:/go/src/vendor/golang_org/x/text/unicode/bidi", ClassVendor},
		{"github.com/go-ole/go-ole", "c:/go/src/github.com/go-ole/go-ole", ClassVendor},
		{"vendor/golang_org/x/text/secure/bidirule", "c:/go/src/vendor/golang_org/x/text/secure/bidirule", ClassVendor},
		{"vendor/golang_org/x/crypto/poly1305", "c:/go/src/vendor/golang_org/x/crypto/poly1305", ClassVendor},
		{"vendor/golang_org/x/crypto/curve25519", "c:/go/src/vendor/golang_org/x/crypto/curve25519", ClassVendor},
		{"github.com/kbinani/screenshot", "c:/go/src/github.com/kbinani/screenshot", ClassVendor},
		{"github.com/go-ole/go-ole/oleutil", "c:/go/src/github.com/go-ole/go-ole/oleutil", ClassVendor},
		{"github.com/shirou/gopsutil/net", ".", ClassUnknown},
		{"github.com/iamacarpet/go-win64api", "c:/go/src/github.com/iamacarpet/go-win64api", ClassVendor},
		{"github.com/kbinani/screenshot/internal/util", "c:/go/src/github.com/kbinani/screenshot/internal/util", ClassVendor},
		{"golang.org/x/sys/windows/svc", "c:/go/src/golang.org/x/sys/windows/svc", ClassVendor},
		{"github.com/shirou/gopsutil/cpu", "c:/go/src/github.com/shirou/gopsutil/cpu", ClassVendor},
		{"github.com/StackExchange/wmi", "c:/go/src/github.com/StackExchange/wmi", ClassVendor},
		{"github.com/shirou/gopsutil/internal/common", "c:/go/src/github.com/shirou/gopsutil/internal/common", ClassVendor},
		{"vendor/golang_org/x/net/idna", "c:/go/src/vendor/golang_org/x/net/idna", ClassVendor},
		{"github.com/lxn/win", "c:/go/src/github.com/lxn/win", ClassVendor},
		{"github.com/shirou/gopsutil/host", "c:/go/src/github.com/shirou/gopsutil/host", ClassVendor},
		{"golang.org/x/sys/windows", "c:/go/src/golang.org/x/sys/windows", ClassVendor},
		{"", "c:/go/src/internal/bytealg", ClassUnknown},
		{"vendor/golang_org/x/net/http/httpguts", "c:/go/src/vendor/golang_org/x/net/http/httpguts", ClassVendor},
		{"vendor/golang_org/x/crypto/internal/chacha20", "c:/go/src/vendor/golang_org/x/crypto/internal/chacha20", ClassVendor},
		{"vendor/golang_org/x/crypto/cryptobyte", "c:/go/src/vendor/golang_org/x/crypto/cryptobyte", ClassVendor},
		{"vendor/golang_org/x/crypto/chacha20poly1305", "c:/go/src/vendor/golang_org/x/crypto/chacha20poly1305", ClassVendor},
		{"golang.org/x/sys/windows/registry", "c:/go/src/golang.org/x/sys/windows/registry", ClassVendor},
		{"vendor/golang_org/x/text/transform", ".", ClassVendor},
		{"golang.org/x/sys/windows/svc/mgr", ".", ClassUnknown},
		{"github.com/shirou/gopsutil/mem", ".", ClassUnknown},
		{"github.com/shirou/gopsutil/process", "c:/go/src/github.com/shirou/gopsutil/process", ClassVendor},
		{"github.com/iamacarpet/go-win64api/shared", "c:/go/src/github.com/iamacarpet/go-win64api/shared", ClassVendor},
		{"vendor/golang_org/x/net/http/httpproxy", "c:/go/src/vendor/golang_org/x/net/http/httpproxy", ClassVendor},
		{"github.com/shirou/w32", "c:/go/src/github.com/shirou/w32", ClassVendor},
	}

	assert := assert.New(t)
	mainPath := "C:/Users/h/CloudStation/Projects/0/ly/lady/src/lady"
	classifier := NewPackageClassifier(mainPath)

	for _, test := range tests {
		t.Run("classify_"+test.pkgsName, func(t *testing.T) {
			pkg := &Package{
				Filepath: test.pkgPath,
				Name:     test.pkgsName,
			}
			class := classifier.Classify(pkg)
			assert.Equal(test.pkgClass, class, fmt.Sprintf("Incorrect classification of: %s with filepath: %s", test.pkgsName, test.pkgPath))
		})
	}
}

func TestGetSourceFiles(t *testing.T) {
	assert := assert.New(t)
	// Setup
	pn := "main"
	fp1 := "main.go"
	fp2 := "extra.go"

	f1 := &Function{Filename: fp1, PackageName: pn, SrcLineStart: 8, Name: "main"}
	f2 := &Function{Filename: fp1, PackageName: pn, SrcLineStart: 15, Name: "start"}
	f3 := &Function{Filename: fp2, PackageName: pn, SrcLineStart: 5, Name: "e1"}
	f4 := &Function{Filename: fp2, PackageName: pn, SrcLineStart: 15, Name: "e2"}

	m1 := &Method{Function: &Function{Filename: fp2, PackageName: pn, SrcLineStart: 10, Name: "m1"}, Receiver: "test"}
	m2 := &Method{Function: &Function{Filename: fp2, PackageName: pn, SrcLineStart: 25, Name: "m2"}, Receiver: "test"}

	pkg := &Package{Name: pn, Filepath: "/",
		Functions: []*Function{f2, f1, f4, f3},
		Methods:   []*Method{m2, m1},
	}

	// Test

	sf := pkg.GetSourceFiles()

	assert.Len(sf, 2, "Should return a count of 2 files")
	assert.Equal(fp2, sf[0].Name, "extra should be sorted first")
	assert.Equal(fp1, sf[1].Name, "main should be sorted last")
}

func TestAthenaCase(t *testing.T) {
	tests := []struct {
		pkgsName string
		pkgPath  string
		pkgClass PackageClass
	}{
		{"Athena/core", "/Users/ruben/golang/src/Athena/core", ClassMain},
		{"Athena/cryptography", "/Users/ruben/golang/src/Athena/cryptography", ClassMain},
		{"Athena/network", "/Users/ruben/golang/src/Athena/network", ClassMain},
		{"Athena/specific", "/Users/ruben/golang/src/Athena/specific", ClassMain},
	}

	assert := assert.New(t)
	mainPath := "/Users/ruben/Programming/Athena"
	classifier := NewPackageClassifier(mainPath)

	for _, test := range tests {
		t.Run("classify_"+test.pkgsName, func(t *testing.T) {
			pkg := &Package{
				Filepath: test.pkgPath,
				Name:     test.pkgsName,
			}
			class := classifier.Classify(pkg)
			assert.Equal(test.pkgClass, class, "Incorrect classification of: "+test.pkgsName)
		})
	}
}

func TestUseGoModVersion(t *testing.T) {
	tests := []struct {
		pkgsName string
		pkgPath  string
		pkgClass PackageClass
	}{
		{"github.com/gohugoio/hugo/watcher", "/root/project/hugo/watcher", ClassMain},
		{"github.com/gohugoio/hugo/cache/filecache", "/root/project/hugo/cache/filecache", ClassMain},
		{"cloud.google.com/go/storage", "/go/pkg/mod/cloud.google.com/go/storage@v1.10.0", ClassVendor},
	}

	assert := assert.New(t)
	mainPath := "/root/project/hugo"
	classifier := NewPackageClassifier(mainPath)

	for _, test := range tests {
		t.Run("classify_"+test.pkgsName, func(t *testing.T) {
			pkg := &Package{
				Filepath: test.pkgPath,
				Name:     test.pkgsName,
			}
			class := classifier.Classify(pkg)
			assert.Equal(test.pkgClass, class, "Incorrect classification of: "+test.pkgsName)
		})
	}
}

func TestCommandLineArgumentsPagkageDetection(t *testing.T) {
	tests := []struct {
		pkgsName string
		pkgPath  string
		pkgClass PackageClass
	}{
		{"x_cgo_sys_thread_creater", ".", ClassSTD},
		{"_cgo_sys_thread_star", ".", ClassSTD},
		{"", ".", ClassGenerated},
		{"gopackage", "gopackage", ClassMain},
		{"gopackage/subpackage", "gopackage/subpackage", ClassMain},
	}

	assert := assert.New(t)
	mainPath := "command-line-arguments"
	classifier := NewPackageClassifier(mainPath)

	for _, test := range tests {
		t.Run("classify_"+test.pkgsName, func(t *testing.T) {
			pkg := &Package{
				Filepath: test.pkgPath,
				Name:     test.pkgsName,
			}
			class := classifier.Classify(pkg)
			assert.Equal(test.pkgClass, class, "Incorrect classification of: "+test.pkgsName)
		})
	}
}

func TestSubSubSubPackage(t *testing.T) {
	tests := []struct {
		pkgsName string
		pkgPath  string
		pkgClass PackageClass
	}{
		{"gopackage", "gopackage", ClassMain},
		{"gopackage/subpackage", "gopackage/subpackage", ClassMain},
	}

	assert := assert.New(t)
	mainPath := "gopackage/cmds/gopackage"
	classifier := NewPackageClassifier(mainPath)

	for _, test := range tests {
		t.Run("classify_"+test.pkgsName, func(t *testing.T) {
			pkg := &Package{
				Filepath: test.pkgPath,
				Name:     test.pkgsName,
			}
			class := classifier.Classify(pkg)
			assert.Equal(test.pkgClass, class, "Incorrect classification of: "+test.pkgsName)
		})
	}
}
