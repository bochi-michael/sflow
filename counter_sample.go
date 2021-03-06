package sflow

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type CounterSample struct {
	SequenceNum      uint32
	SourceIdType     byte
	SourceIdIndexVal uint32 // NOTE: this is 3 bytes in the datagram
	numRecords       uint32
	Records          []Record
}

func (s CounterSample) String() string {
	type X CounterSample
	x := X(s)
	return fmt.Sprintf("CounterSample: %+v", x)
}

// SampleType returns the type of sFlow sample.
func (s *CounterSample) SampleType() int {
	return TypeCounterSample
}

func (s *CounterSample) GetRecords() []Record {
	return s.Records
}

func decodeCounterSample(r io.ReadSeeker) (Sample, error) {
	s := &CounterSample{}

	var err error

	err = binary.Read(r, binary.BigEndian, &s.SequenceNum)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &s.SourceIdType)
	if err != nil {
		return nil, err
	}

	var srcIdIndexVal [3]byte
	n, err := r.Read(srcIdIndexVal[:])
	if err != nil {
		return nil, err
	}

	if n != 3 {
		return nil, errors.New("sflow: counter sample decoding error")
	}

	s.SourceIdIndexVal = uint32(srcIdIndexVal[2]) |
		uint32(srcIdIndexVal[1]<<8) |
		uint32(srcIdIndexVal[0]<<16)

	err = binary.Read(r, binary.BigEndian, &s.numRecords)
	if err != nil {
		return nil, err
	}

	for i := uint32(0); i < s.numRecords; i++ {
		var rec Record
		rec, err := decodeCounterRecord(r)
		if err != nil {
			return nil, err
		} else {
			s.Records = append(s.Records, rec)
		}
	}
	return s, nil
}

func (s *CounterSample) encode(w io.Writer) error {
	var err error

	// We first need to encode the records.
	buf := &bytes.Buffer{}

	for _, rec := range s.Records {
		err = rec.encode(buf)
		if err != nil {
			return ErrEncodingRecord
		}
	}

	// Fields
	encodedSampleSize := uint32(4 + 1 + 3 + 4)

	// Encoded records
	encodedSampleSize += uint32(buf.Len())

	err = binary.Write(w, binary.BigEndian, uint32(s.SampleType()))
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, encodedSampleSize)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.SequenceNum)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian,
		uint32(s.SourceIdType)|s.SourceIdIndexVal<<24)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian,
		uint32(len(s.Records)))
	if err != nil {
		return err
	}

	_, err = io.Copy(w, buf)
	return err
}
