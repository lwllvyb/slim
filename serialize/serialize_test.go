package serialize

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"testing"

	"github.com/openacid/slim/array"
	"github.com/openacid/slim/version"
)

var testDataFn = "data"

func TestBinaryWriteUint64Length(t *testing.T) {
	expectSize := 8

	for _, u64 := range []uint64{0, math.MaxUint64 / 2, math.MaxUint64} {
		writer := new(bytes.Buffer)

		err := binary.Write(writer, binary.LittleEndian, u64)
		if err != nil {
			t.Fatalf("failed to write uint64: %v", err)
		}

		buf := writer.Bytes()

		if len(buf) != expectSize {
			t.Fatalf("uint64 does not take %d bytes", expectSize)
		}
	}
}

func TestBytesToString(t *testing.T) {

	var str string = bytesToString(nil, 0)
	if str != "" {
		t.Fatalf("failed to handle nil: %s", str)
	}

	str = bytesToString([]byte{}, 0)
	if str != "" {
		t.Fatalf("failed to handle nil: %s", str)
	}

	str = bytesToString([]byte{'a', 'b', 'c'}, 0)
	if str != "abc" || len(str) != 3 {
		t.Fatalf("failed to handle abc: %s", str)
	}

	str = bytesToString([]byte{'1', '.', '0', '.', '0', 0}, 0)
	if str != "1.0.0" || len(str) != 5 {
		t.Fatalf("failed to handle 1.0.0'0': %s", str)
	}

	bBuf := []byte{'1', '.', '0', '.', '0', 0}
	str = bytesToString(bBuf, 0)

	bBuf[0] = '2'
	if str != "1.0.0" {
		t.Fatalf("wrong str value after modify byte buffer: %s", str)
	}
}

func TestMakeDataHeader(t *testing.T) {
	ver := "0.0.1"
	dataSize := uint64(1000)
	headerSize := uint64(100)
	header := makeDataHeader(ver, headerSize, dataSize)

	if header.DataSize != dataSize {
		t.Fatalf("wrong data size")
	}

	if header.HeaderSize != headerSize {
		t.Fatalf("wrong header size")
	}

	verStr := bytesToString(header.Version[:], 0)
	if verStr != ver {
		t.Fatalf("wrong version: %s, expect: %s", verStr, ver)
	}

	header = makeDefaultDataHeader(dataSize)
	if header.DataSize != dataSize {
		t.Fatalf("wrong data size")
	}

	// sizeof(uint64) * 2 + version.MAXLEN
	if header.HeaderSize != 32 {
		t.Fatalf("wrong header size: %v", header.HeaderSize)
	}

	if len(header.Version) != version.MAXLEN {
		t.Fatalf("wrong version length: %v", len(header.Version))
	}

	verStr = bytesToString(header.Version[:], 0)
	if verStr != version.VERSION {
		t.Fatalf("wrong version: %s, expect: %s", verStr, version.VERSION)
	}
}

func TestMarshalUnMarshalHeader(t *testing.T) {
	// marshal
	wOFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	writer, err := os.OpenFile(testDataFn, wOFlags, 0755)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer os.Remove(testDataFn)

	sHeader := makeDefaultDataHeader(1000)

	gHeaderSize := GetMarshalHeaderSize()
	if gHeaderSize != 32 {
		t.Fatalf("wrong header size: 32, %d", gHeaderSize)
	}

	err = marshalHeader(writer, sHeader)
	if err != nil {
		t.Fatalf("failed to marshalHeader: %v", err)
	}

	writer.Close()

	// unmarshal
	reader, err := os.OpenFile(testDataFn, os.O_RDONLY, 0755)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer reader.Close()

	rSHeader, err := UnmarshalHeader(reader)
	if err != nil {
		t.Fatalf("failed to unmarshalHeader: %v", err)
	}

	if rSHeader.DataSize != sHeader.DataSize {
		t.Fatalf("wrong data size: %v, %v", rSHeader.DataSize, sHeader.DataSize)
	}

	if rSHeader.HeaderSize != sHeader.HeaderSize {
		t.Fatalf("wrong header size: %v, %v",
			rSHeader.HeaderSize, sHeader.HeaderSize)
	}

	for idx, sByte := range sHeader.Version {
		rByte := rSHeader.Version[idx]
		if rByte != sByte {
			t.Fatalf("wrong byte in version: %v, %v, %v", idx, sByte, rByte)
		}
	}

	rVersion := bytesToString(rSHeader.Version[:], 0)
	if rVersion != version.VERSION {
		t.Fatalf("wrong version string: %v, %v", rVersion, version.VERSION)
	}
}

func TestMarshalUnMarshal(t *testing.T) {
	index := []int32{10, 20, 30, 40, 50, 60}
	elts := []uint32{10, 20, 30, 40, 50, 60}

	a, err := array.New(index, elts)
	if err != nil {
		t.Fatalf("failed to init compacted array: %+v", err)
	}

	marshalSize := GetMarshalSize(a)

	// marshal
	wOFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	writer, err := os.OpenFile(testDataFn, wOFlags, 0755)
	if err != nil {
		t.Fatalf("failed to create file: %s, %v", testDataFn, err)
	}
	defer os.Remove(testDataFn)

	cnt, err := Marshal(writer, a)
	if err != nil {
		t.Fatalf("failed to store compacted array: %v", err)
	}

	writer.Close()

	fInfo, err := os.Stat(testDataFn)
	if err != nil {
		t.Fatalf("failed to get file info: %s, %v", testDataFn, err)
	}

	if fInfo.Size() != cnt {
		t.Fatalf("wrong file size: %d, %d", fInfo.Size(), cnt)
	}

	if fInfo.Size() != marshalSize {
		t.Fatalf("wrong marshal size: %d, %d", fInfo.Size(), marshalSize)
	}

	// unmarshal
	reader, err := os.OpenFile(testDataFn, os.O_RDONLY, 0755)
	if err != nil {
		t.Fatalf("failed to read file: %s, %v", testDataFn, err)
	}
	defer reader.Close()

	a2, err := array.NewEmpty(uint32(0))
	if err != nil {
		t.Fatalf("expected no error but: %+v", err)
	}

	err = Unmarshal(reader, a2)
	if err != nil {
		t.Fatalf("failed to load data: %v", err)
	}

	// check compacted array
	checkCompactedArray(index, a2, a, t)
}

type IncomleteReaderWriter struct {
	Buf []byte
}

func (rw *IncomleteReaderWriter) Read(p []byte) (n int, err error) {
	// Read 1 byte per Read()
	b := rw.Buf[0]
	rw.Buf = rw.Buf[1:]
	p[0] = b
	return 1, nil
}

func (rw *IncomleteReaderWriter) Write(p []byte) (n int, err error) {
	rw.Buf = append(rw.Buf, p...)
	return len(p), nil
}

func TestUnMarshalFromIncompleteReader(t *testing.T) {

	// Marshal() must work correctly
	// with an io.Reader:Read(p []byte)
	// returns n < len(p).

	index := []int32{10, 20, 30, 40, 50, 60}
	elts := []uint32{10, 20, 30, 40, 50, 60}

	a1, err := array.New(index, elts)
	if err != nil {
		t.Fatalf("failed to init compacted array")
	}

	marshalSize := GetMarshalSize(a1)

	rw := &IncomleteReaderWriter{}

	// marshal

	cnt, err := Marshal(rw, a1)
	if err != nil {
		t.Fatalf("failed to store compacted array: %v", err)
	}
	if cnt != marshalSize {
		t.Fatalf("byte written %d != expected size %d", cnt, marshalSize)
	}

	// unmarshal

	a2, err := array.NewEmpty(uint32(0))
	if err != nil {
		t.Fatalf("expected no error but: %+v", err)
	}

	err = Unmarshal(rw, a2)
	if err != nil {
		t.Fatalf("failed to load data: %v", err)
	}

	// check compacted array
	checkCompactedArray(index, a2, a1, t)
}

func TestMarshalAtUnMarshalAt(t *testing.T) {
	index1 := []int32{10, 20, 30, 40, 50, 60}
	elts1 := []uint32{10, 20, 30, 40, 50, 60}
	index2 := []int32{15, 25, 35, 45, 55, 65}
	elts2 := []uint32{15, 25, 35, 45, 55, 65}

	sArray1, err := array.New(index1, elts1)
	if err != nil {
		t.Fatalf("failed to init compacted array: %+v", err)
	}
	sArray2, err := array.New(index2, elts2)
	if err != nil {
		t.Fatalf("failed to init compacted array: %+v", err)
	}

	// marshalat
	wOFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	writer, err := os.OpenFile(testDataFn, wOFlags, 0755)
	if err != nil {
		t.Fatalf("failed to create file: %s, %v", testDataFn, err)
	}
	defer os.Remove(testDataFn)

	offset1 := int64(0)
	cnt, err := MarshalAt(writer, offset1, sArray1)
	if err != nil {
		t.Fatalf("failed to store compacted array: %v", err)
	}

	offset2 := offset1 + cnt
	_, err = MarshalAt(writer, offset2, sArray2)
	if err != nil {
		t.Fatalf("failed to store compacted array: %v", err)
	}

	writer.Close()

	// unmarshalat
	reader, err := os.OpenFile(testDataFn, os.O_RDONLY, 0755)
	if err != nil {
		t.Fatalf("failed to read file: %s, %v", testDataFn, err)
	}
	defer reader.Close()

	rSArray1, err := array.NewEmpty(uint32(0))
	if err != nil {
		t.Fatalf("expected no error but: %+v", err)
	}
	_, err = UnmarshalAt(reader, offset1, rSArray1)
	if err != nil {
		t.Fatalf("failed to load data: %v", err)
	}

	checkCompactedArray(index1, rSArray1, sArray1, t)

	rSArray2, err := array.NewEmpty(uint32(0))
	if err != nil {
		t.Fatalf("expected no error but: %+v", err)
	}
	_, err = UnmarshalAt(reader, offset2, rSArray2)
	if err != nil {
		t.Fatalf("failed to load data: %v", err)
	}

	checkCompactedArray(index2, rSArray2, sArray2, t)
}

func checkCompactedArray(index []int32, a1, a2 *array.Array, t *testing.T) {
	if a1.Cnt != a2.Cnt {
		t.Fatalf("wrong Cnt: %d, %d", a1.Cnt, a2.Cnt)
	}

	if len(a2.Bitmaps) != len(a1.Bitmaps) {
		t.Fatalf("wrong bitmap len: %d, %d", len(a1.Bitmaps), len(a2.Bitmaps))
	}

	for idx, elt := range a2.Bitmaps {
		if a1.Bitmaps[idx] != elt {
			t.Fatalf("wrong bitmap value: %v, %v", a1.Bitmaps[idx], elt)
		}
	}

	if len(a2.Offsets) != len(a1.Offsets) {
		t.Fatalf("wrong offset len: %v, %v", a1.Offsets, a2.Offsets)
	}

	for idx, elt := range a2.Offsets {
		if a1.Offsets[idx] != elt {
			t.Fatalf("wrong offsets value: %v, %v", a1.Offsets[idx], elt)
		}
	}

	if len(a2.Elts) != len(a1.Elts) {
		t.Fatalf("wrong Elts len: %v, %v", a1.Elts, a2.Elts)
	}

	for _, idx := range index {
		a, _ := a2.Get(idx)
		b, _ := a1.Get(idx)
		sVal := a.(uint32)
		rsVal := b.(uint32)

		if sVal != rsVal || sVal != uint32(idx) {
			t.Fatalf("wrong Elts value: %v, %v, %v", sVal, rsVal, idx)
		}
	}
}

type testWriterReader struct {
	b [512]byte
}

func (t *testWriterReader) WriteAt(b []byte, off int64) (n int, err error) {
	length := len(b)
	for i := 0; i < length; i++ {
		t.b[int64(i)+off] = b[i]
	}
	return length, nil
}

func (t *testWriterReader) ReadAt(b []byte, off int64) (n int, err error) {
	length := len(b)
	copy(b, t.b[off:off+int64(length)])
	return length, nil
}

func TestWriteAtReadAt(t *testing.T) {
	rw := &testWriterReader{}
	index1 := []int32{10, 20, 30, 40, 50, 60}
	elts := []uint32{10, 20, 30, 40, 50, 60}

	a1, err := array.New(index1, elts)
	if err != nil {
		t.Fatalf("failed to init compacted array: %+v", err)
	}

	_, err = MarshalAt(rw, 10, a1)
	if err != nil {
		t.Fatalf("failed to store compacted array: %v", err)
	}

	a2, err := array.NewEmpty(uint32(0))
	if err != nil {
		t.Fatalf("expected no error but: %+v", err)
	}
	_, err = UnmarshalAt(rw, 10, a2)
	if err != nil {
		t.Fatalf("failed to load data: %+v", err)
	}

	checkCompactedArray(index1, a2, a1, t)
}
