package rpc

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type RpcHandler interface {
	setBody(data []byte)
	getBody() []byte
	out(data []byte)
	err(code, data string)
	validate() (msg string, ok bool)
}
type rpcJson struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

func (rj *rpcJson) Get(key string) (interface{}, bool) {
	v, b := rj.Params[key]
	return v, b
}
func (rj *rpcJson) Type(key string) string {
	v, b := rj.Get(key)
	if !b {
		return ""
	}
	return reflect.TypeOf(v).String()
}
func (rj *rpcJson) VerifyType(key, types string) bool {
	if rj.Type(key) == types {
		return true
	} else {
		return false
	}

}
func parseJson(jsonb []byte) (*rpcJson, error) {
	var rpcjson rpcJson
	err := json.Unmarshal(jsonb, &rpcjson)
	//fmt.Printf("%+v\n", rpcjson)
	return &rpcjson, err
}
func Route(rh RpcHandler) (res []byte, err error) {
	data := rh.getBody()
	rj, err := parseJson(data)
	if err != nil {
		fmt.Println(err)
	}
	hd, ok := rpcHandler[rj.Method]
	if ok {
		res, err = hd(rj)
	} else {
		res, err = errcode(NoMethod, rj.Method)
	}
	return
}
