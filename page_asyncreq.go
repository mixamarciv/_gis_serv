package main

import (
	"net/http"

	"strings"

	//"github.com/parnurzeal/gorequest"

	"fmt"

	mf "github.com/mixamarciv/gofncstd3000"

	"errors"

	"io/ioutil"

	"strconv"

	//"github.com/go-ini/ini"
	//"os/exec"

	_ "github.com/nakagami/firebirdsql"
)

func init() {
	rtr.HandleFunc("/asyncreq", mf.LogreqF("/asyncreq", post_asyncreq)).Methods("POST")
	fmt.Printf("")
}

//отправляем запрос
func post_asyncreq(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	//checkErrorJSON("error body: \n"+string(body[:]), errors.New("len(body)=="+strconv.Itoa(len(body))), w)

	if checkErrorJSON("error: ioutil.ReadAll(r.Body) ", err, w) {
		return
		checkErrorJSON("error body: \n"+string(body[:]), errors.New("len(body)=="+strconv.Itoa(len(body))), w)
	}

	jsonbody, err := mf.FromJson([]byte(body))
	if checkErrorJSON("FromJson error: body "+string(body), err, w) {
		return
	}

	xml := jsonbody["xml"].(string)
	data := jsonbody["data"].(string)

	if len(data) == 0 {
		s := fmt.Sprintf("%+v", r.Body)
		checkErrorJSON("error no data: \n"+s, errors.New("len(data)==0"), w)
		return
	}

	json, err := mf.FromJson([]byte(data))
	if checkErrorJSON("FromJson error", err, w) {
		return
	}

	signelem, ok := json["signelem"].(string)
	if !ok {
		signelem = "ahz"
		return
	}

	datetime = mf.CurTimeStrShort()
	path := mf.AppPath() + "/temp_asyncreq/" + datetime[0:8] + "/" + datetime[9:11]
	path = strings.Replace(path, "\\", "/", -1)
	mf.MkdirAll(path)

	file := path + "/" + mf.StrRegexpReplace(signelem, "[: \\[\\]\\?\\(\\)\\&\\%\"'`]", "-") + "_" + mf.CurTimeStrShort()
	mf.FileWriteStr(file+".xml", xml)
	mf.FileWriteStr(file+".data", data)

	var ret []string
	ret = append(ret, xml_sign)
	ret = append(ret, outstr)

	json_ret, err := mf.ToJson(ret)
	if checkErrorJSON("ToJson error", err, w) {
		return
	}

	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	w.Write(json_ret)
}
