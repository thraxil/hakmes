package main

import (
	"errors"
	"strings"
)

type Key struct {
	Algorithm string
	Value     []byte
}

func KeyFromString(str string) (*Key, error) {
	parts := strings.Split(str, ":")
	algorithm := parts[0]
	if algorithm != "sha1" {
		return nil, errors.New("can only handle sha1 now")
	}
	str = parts[1]
	if len(str) != 40 {
		return nil, errors.New("invalid key")
	}
	return &Key{algorithm, []byte(str)}, nil
}

func (k Key) String() string {
	return k.Algorithm + ":" + string(k.Value)
}

func (k Key) Valid() bool {
	return k.Algorithm == "sha1" && len(k.String()) == 40
}
