module github.com/jonathongardner/virtualfs

go 1.23.5

toolchain go1.23.8

require (
	github.com/gabriel-vasile/mimetype v1.4.4
	github.com/google/uuid v1.6.0
	github.com/jonathongardner/fifo v0.0.0-20250417191342-caa2a331a62d
)

require golang.org/x/net v0.25.0 // indirect

replace github.com/jonathongardner/fifo => ../../../Projects/fifo
