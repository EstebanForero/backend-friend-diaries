package storage

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestProbeVideoDurationMP4(t *testing.T) {
	mvhd := make([]byte, 20)
	binary.BigEndian.PutUint32(mvhd[12:16], 1_000)
	binary.BigEndian.PutUint32(mvhd[16:20], 179_500)
	data := append(mp4Box("ftyp", []byte("isom0000")), mp4Box("moov", mp4Box("mvhd", mvhd))...)

	duration, err := ProbeVideoDuration(data, "video/mp4")
	if err != nil {
		t.Fatalf("ProbeVideoDuration returned error: %v", err)
	}
	if math.Abs(duration-179.5) > 0.001 {
		t.Fatalf("duration = %v, want 179.5", duration)
	}
}

func TestProbeVideoDurationWebM(t *testing.T) {
	duration := make([]byte, 8)
	binary.BigEndian.PutUint64(duration, math.Float64bits(180_000))
	data := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x80}
	data = append(data, []byte{0x2a, 0xd7, 0xb1, 0x83, 0x0f, 0x42, 0x40}...)
	data = append(data, []byte{0x44, 0x89, 0x88}...)
	data = append(data, duration...)

	seconds, err := ProbeVideoDuration(data, "video/webm")
	if err != nil {
		t.Fatalf("ProbeVideoDuration returned error: %v", err)
	}
	if math.Abs(seconds-180) > 0.001 {
		t.Fatalf("duration = %v, want 180", seconds)
	}
}

func TestProbeVideoDurationRejectsUnsupportedContainer(t *testing.T) {
	if _, err := ProbeVideoDuration([]byte("not-a-video"), "video/mp4"); err == nil {
		t.Fatal("expected invalid MP4 to be rejected")
	}
}

func mp4Box(kind string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[:4], uint32(len(box)))
	copy(box[4:8], kind)
	copy(box[8:], payload)
	return box
}
