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
	"os/exec"
)

func init() {
	rtr.HandleFunc("/sign", mf.LogreqF("/sign", post_sign)).Methods("POST")
	fmt.Printf("")
}

//отправляем запрос
func post_sign(w http.ResponseWriter, r *http.Request) {

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
	/***
		if data[0:1] != "{" {
			data = "{" + data + "}"
		}
	****/
	json, err := mf.FromJson([]byte(data))
	if checkErrorJSON("FromJson error", err, w) {
		return
	}

	signelem, ok := json["signelem"].(string)
	if !ok {
		checkErrorJSON("error: param signelem not found", errors.New(" signelem not found"), w)
		return
	}
	{
		i := strings.Index(xml, signelem)
		if i == -1 {
			checkErrorJSON("error: signelem:"+signelem+" not found in xml", errors.New(" signelem:"+signelem+" not found in xml"), w)
			return
		}
	}
	xml = strings.Replace(xml, signelem, signelem+` Id="signed_data_container"`, 1)

	signname, ok := json["signname"].(string)
	if !ok {
		checkErrorJSON("error: param signname not found", errors.New(" signname not found"), w)
		return
	}
	exec_file := mf.AppPath() + "/xades/" + signname + "/xades-demo.exe"
	exec_file = strings.Replace(exec_file, "\\", "/", -1)
	/*********
		cfg, err := ini.Load(mf.AppPath() + "/certs.ini")
		if checkErrorJSON("ERROR read ini file certs.ini", err, w) {
			return
		}

		inis, err := cfg.GetSection("")
		if checkErrorJSON("ERROR get section in certs.ini", err, w) {
			return
		}

		signval := inis.Key(signname).MustString(signname + "_not_found")
	**********/

	path := mf.AppPath() + "/temp_sign/" + mf.CurTimeStrShort()[0:8]
	path = strings.Replace(path, "\\", "/", -1)
	mf.MkdirAll(path)

	file := path + "/" + mf.StrRegexpReplace(signelem, "[: \\[\\]\\?\\(\\)\\&\\%\"'`]", "-") + "_" + mf.CurTimeStrShort() + ".xml"
	mf.FileWriteStr(file, xml)

	cmd := `"` + exec_file + `" sign -f"` + file + `" -o"` + file + `.sign" -esigned_data_container -p123`
	out, err := exec.Command(exec_file, "sign", `-f`+file+``, `-o`+file+`.sign`, "-esigned_data_container", "-p123").Output()
	if checkErrorJSON("error: execute cmd: "+cmd, err, w) {
		return
	}

	outstr := mf.StrTr(string(out), "cp866", "UTF-8")
	{
		i := strings.Index(outstr, "Файл успешно подписан")
		if i == -1 {
			checkErrorJSON("error sign: "+cmd, errors.New(" sign ERROR: \ncmd: "+cmd+"\n\nout: "+outstr), w)
			return
		}
	}

	xml_sign, err := mf.FileReadStr(file + ".sign")
	if checkErrorJSON("error: read signed file: "+file+".sign", err, w) {
		return
	}

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
