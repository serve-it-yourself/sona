package sona

import "errors"

var errAddrEmpty = errors.New("addr is empty")

func IsAddrEmptyErr(err error) bool {
	return errors.Is(err, errAddrEmpty)
}
