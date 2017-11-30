package timestamp

import "testing"

func TestTimestamp(t *testing.T) {
	ts := Now()
	buf, _ := ts.MarshalBinary()
	ts2 := new(Timestamp)
	ts2.UnmarshalBinary(buf)

	if int64(ts) != int64(*ts2) {
		t.Errorf("unexpected result: got %d, want %d", ts2, ts)
	}
}
