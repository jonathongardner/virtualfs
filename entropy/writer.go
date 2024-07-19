package entropy

import (
	"math"
)

// an io.Wirter like object for calculating the entropy of a file
type Writer struct {
	// Write a slice of bytes to the entropy writer
	frequency [256]uint64
	count     uint64
}

func NewWriter() *Writer {
	return &Writer{frequency: [256]uint64{}}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	for _, b := range p {
		w.frequency[b]++
	}
	size := len(p)
	w.count += uint64(size)
	return size, nil
}

func (w *Writer) Entropy() float64 {
	ent := float64(0.0)
	size := float64(w.count)
	for i := 0; i < 256; i++ {
		if w.frequency[i] > 0 {
			p := float64(w.frequency[i]) / size
			ent -= p * math.Log2(p)
		}
	}
	return ent
}
