package rpc

import (
	"fmt"
	"strconv"
)

const (
	Success   = 2000
	NoMethod  = 4001
	TypeWrong = 5001
	NoField   = 5002
	Nomarl    = 5003
)

var codes = map[int]string{
	NoMethod:  "no method",
	TypeWrong: "type wrong",
	NoField:   "no field",
	Nomarl:    "",
}

func errcode(code int, p ...string) (res []byte, err error) {
	res = []byte(strconv.Itoa(code))
	c, ok := codes[code]
	if ok {
		if len(p) > 0 {
			err = fmt.Errorf(p[0]+": %v", c)
		} else {
			err = fmt.Errorf("%v", c)
		}
	}
	return
}
