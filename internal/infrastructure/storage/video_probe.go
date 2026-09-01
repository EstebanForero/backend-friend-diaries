package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// ProbeVideoDuration reads duration metadata directly from the uploaded MP4 or
// WebM container. It intentionally does not trust a duration supplied by the
// client, because the duration is part of the server-side premium upload rules.
func ProbeVideoDuration(data []byte, mimeType string) (float64, error) {
	switch mimeType {
	case "video/mp4":
		return probeMP4Duration(data)
	case "video/webm":
		return probeWebMDuration(data)
	default:
		return 0, fmt.Errorf("unsupported video type: %s", mimeType)
	}
}

func probeMP4Duration(data []byte) (float64, error) {
	if len(data) < 12 || !bytes.Contains(data[:min(len(data), 64)], []byte("ftyp")) {
		return 0, errors.New("invalid MP4 container")
	}

	var walk func([]byte) (float64, bool)
	walk = func(buf []byte) (float64, bool) {
		for offset := 0; offset+8 <= len(buf); {
			size := uint64(binary.BigEndian.Uint32(buf[offset : offset+4]))
			kind := string(buf[offset+4 : offset+8])
			header := uint64(8)
			if size == 1 {
				if offset+16 > len(buf) {
					return 0, false
				}
				size = binary.BigEndian.Uint64(buf[offset+8 : offset+16])
				header = 16
			} else if size == 0 {
				size = uint64(len(buf) - offset)
			}
			if size < header || uint64(offset)+size > uint64(len(buf)) {
				return 0, false
			}
			payload := buf[uint64(offset)+header : uint64(offset)+size]
			if kind == "mvhd" {
				if seconds, ok := parseMVHD(payload); ok {
					return seconds, true
				}
			}
			if kind == "moov" {
				if seconds, ok := walk(payload); ok {
					return seconds, true
				}
			}
			offset += int(size)
		}
		return 0, false
	}

	if duration, ok := walk(data); ok && duration > 0 && !math.IsInf(duration, 0) && !math.IsNaN(duration) {
		return duration, nil
	}
	return 0, errors.New("MP4 duration metadata not found")
}

func parseMVHD(payload []byte) (float64, bool) {
	if len(payload) < 20 {
		return 0, false
	}
	if payload[0] == 1 {
		if len(payload) < 32 {
			return 0, false
		}
		timescale := binary.BigEndian.Uint32(payload[20:24])
		duration := binary.BigEndian.Uint64(payload[24:32])
		if timescale == 0 {
			return 0, false
		}
		return float64(duration) / float64(timescale), true
	}
	timescale := binary.BigEndian.Uint32(payload[12:16])
	duration := binary.BigEndian.Uint32(payload[16:20])
	if timescale == 0 {
		return 0, false
	}
	return float64(duration) / float64(timescale), true
}

func probeWebMDuration(data []byte) (float64, error) {
	if len(data) < 8 || !bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return 0, errors.New("invalid WebM container")
	}

	timecodeScale := uint64(1_000_000) // Matroska default, in nanoseconds.
	if value, ok := findEBMLUnsigned(data, []byte{0x2a, 0xd7, 0xb1}); ok && value > 0 {
		timecodeScale = value
	}

	var durationUnits float64
	if raw, ok := findEBMLPayload(data, []byte{0x44, 0x89}); ok {
		switch len(raw) {
		case 4:
			durationUnits = float64(math.Float32frombits(binary.BigEndian.Uint32(raw)))
		case 8:
			durationUnits = math.Float64frombits(binary.BigEndian.Uint64(raw))
		}
	}

	// MediaRecorder WebM output does not always include an Info/Duration field.
	// In that case, derive it from the last cluster and block timecodes.
	if clusterUnits := lastWebMTimecode(data); clusterUnits > durationUnits {
		durationUnits = clusterUnits
	}

	seconds := durationUnits * float64(timecodeScale) / 1_000_000_000
	if seconds <= 0 || math.IsInf(seconds, 0) || math.IsNaN(seconds) {
		return 0, errors.New("WebM duration metadata not found")
	}
	return seconds, nil
}

func findEBMLUnsigned(data, id []byte) (uint64, bool) {
	payload, ok := findEBMLPayload(data, id)
	if !ok || len(payload) == 0 || len(payload) > 8 {
		return 0, false
	}
	var value uint64
	for _, b := range payload {
		value = value<<8 | uint64(b)
	}
	return value, true
}

func findEBMLPayload(data, id []byte) ([]byte, bool) {
	for start := 0; start < len(data); {
		i := bytes.Index(data[start:], id)
		if i < 0 {
			return nil, false
		}
		pos := start + i + len(id)
		size, n, unknown, ok := readVInt(data[pos:])
		if ok && !unknown && size <= uint64(len(data)-pos-n) {
			return data[pos+n : pos+n+int(size)], true
		}
		start += i + 1
	}
	return nil, false
}

func lastWebMTimecode(data []byte) float64 {
	clusterID := []byte{0x1f, 0x43, 0xb6, 0x75}
	positions := make([]int, 0, 8)
	for start := 0; start < len(data); {
		i := bytes.Index(data[start:], clusterID)
		if i < 0 {
			break
		}
		positions = append(positions, start+i)
		start += i + len(clusterID)
	}

	var maximum int64
	for index, position := range positions {
		bodyStart := position + len(clusterID)
		_, sizeLen, _, ok := readVInt(data[bodyStart:])
		if !ok {
			continue
		}
		bodyStart += sizeLen
		bodyEnd := len(data)
		if index+1 < len(positions) {
			bodyEnd = positions[index+1]
		}
		cluster := data[bodyStart:bodyEnd]
		clusterTime := int64(0)
		if value, ok := findEBMLUnsigned(cluster[:min(len(cluster), 256)], []byte{0xe7}); ok {
			clusterTime = int64(value)
		}
		if clusterTime > maximum {
			maximum = clusterTime
		}

		for start := 0; start < len(cluster); {
			i := bytes.IndexByte(cluster[start:], 0xa3) // SimpleBlock
			if i < 0 {
				break
			}
			pos := start + i + 1
			size, n, unknown, ok := readVInt(cluster[pos:])
			if !ok || unknown || size > uint64(len(cluster)-pos-n) {
				start += i + 1
				continue
			}
			block := cluster[pos+n : pos+n+int(size)]
			_, trackLen, _, trackOK := readVInt(block)
			if trackOK && len(block) >= trackLen+2 {
				relative := int64(int16(binary.BigEndian.Uint16(block[trackLen : trackLen+2])))
				if value := clusterTime + relative; value > maximum {
					maximum = value
				}
			}
			start = pos + n + int(size)
		}
	}
	return float64(maximum)
}

func readVInt(data []byte) (value uint64, length int, unknown bool, ok bool) {
	if len(data) == 0 || data[0] == 0 {
		return 0, 0, false, false
	}
	mask := byte(0x80)
	length = 1
	for length <= 8 && data[0]&mask == 0 {
		mask >>= 1
		length++
	}
	if length > 8 || len(data) < length {
		return 0, 0, false, false
	}
	value = uint64(data[0] &^ mask)
	for i := 1; i < length; i++ {
		value = value<<8 | uint64(data[i])
	}
	unknownValue := uint64(1)<<(7*length) - 1
	return value, length, value == unknownValue, true
}
