module github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip

replace github.com/hit9/bitproto/lib/go => ../../../../../lib/go

replace github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/bp => ./bp

replace github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/framebp => ./framebp

go 1.15

require (
	github.com/hit9/bitproto/lib/go v0.0.0-00010101000000-000000000000 // indirect
	github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/bp v0.0.0-00010101000000-000000000000
	github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/framebp v0.0.0-00010101000000-000000000000
)
