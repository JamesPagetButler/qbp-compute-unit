module github.com/JamesPagetButler/qbp-visualizer

go 1.25.0

// LOCAL SETUP REQUIRED: kaiju (native game engine) is not available on the
// Go module proxy. Before building this module, add to go.mod locally:
//   replace kaiju => /path/to/kaiju/engine/src
// Do NOT commit machine-specific replace directives.
// CI build is deferred until kaiju publishes to a module proxy.
require (
	github.com/JamesPagetButler/qbp-compute-unit/emulator v0.1.0-rc1
	kaiju v0.0.0-00010101000000-000000000000
)

require (
	github.com/KaijuEngine/uuid v1.0.0 // indirect
	github.com/tdewolff/parse/v2 v2.8.1 // indirect
	golang.design/x/clipboard v0.7.1 // indirect
	golang.org/x/exp/shiny v0.0.0-20250819193227-8b4c13bb791b // indirect
	golang.org/x/image v0.30.0 // indirect
	golang.org/x/mobile v0.0.0-20250813145510-f12310a0cfd9 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)
