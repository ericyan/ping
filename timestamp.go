package ping

import "time"

// A Timestamp represents a point in time as the number of nanoseconds
// elapsed since January 1, 1970 UTC.
type Timestamp int64

// Now returns the current timestamp.
func Now() Timestamp {
	return Timestamp(time.Now().UnixNano())
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (t Timestamp) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	for i := uint(0); i < 8; i++ {
		buf[i] = byte((t >> ((7 - i) * 8)) & 0xff)
	}
	return buf, nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (t *Timestamp) UnmarshalBinary(data []byte) error {
	var nsec int64
	for i := uint(0); i < 8; i++ {
		nsec += int64(data[i]) << ((7 - i) * 8)
	}
	*t = Timestamp(nsec)
	return nil
}
