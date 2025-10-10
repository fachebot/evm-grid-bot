package utils

import (
	"bytes"
	"encoding/binary"

	"github.com/speps/go-hashids/v2"
)

type HashEncoder struct {
	hashId *hashids.HashID
}

func NewHashEncoder(salt string) (*HashEncoder, error) {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 32

	hashId, err := hashids.NewWithData(hd)
	if err != nil {
		return nil, err
	}
	return &HashEncoder{hashId: hashId}, nil
}

func (coder *HashEncoder) Decryption(message string) (string, error) {
	numbers, err := coder.hashId.DecodeInt64WithError(message)
	if err != nil {
		return "", err
	}
	return coder.int64ArrayToString(numbers), nil
}

func (coder *HashEncoder) Encryption(message string) (string, error) {
	numbers := coder.stringToInt64Array(message)
	return coder.hashId.EncodeInt64(numbers)
}

func (coder *HashEncoder) stringToInt64Array(str string) []int64 {
	if str == "" {
		return nil
	}

	data := []byte(str)
	numChunks := (len(data) + 7) / 8 // 向上取整
	result := make([]int64, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * 8
		end := start + 8

		temp := make([]byte, 8)

		if start < len(data) {
			if end > len(data) {
				end = len(data)
			}
			copy(temp, data[start:end])
		}

		result[i] = int64(binary.LittleEndian.Uint64(temp))
	}

	return result
}

func (coder *HashEncoder) int64ArrayToString(int64Array []int64) string {
	if len(int64Array) == 0 {
		return ""
	}

	data := make([]byte, len(int64Array)*8)
	for i, val := range int64Array {
		binary.LittleEndian.PutUint64(data[i*8:], uint64(val))
	}

	return string(bytes.TrimRight(data, "\u0000"))
}
