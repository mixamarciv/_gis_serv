package main

import (
	mf "github.com/mixamarciv/gofncstd3000"

	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	//"github.com/go-ini/ini"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"

	xmlx "github.com/jteeuwen/go-pkg-xmlx"

	"database/sql"

	_ "github.com/nakagami/firebirdsql"
)

func init() {
	rtr.HandleFunc("/asyncreq", mf.LogreqF("/asyncreq", post_asyncreq)).Methods("POST")
	log.Printf("")

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

	xml, data, err = make_asynqreq(xml, data)
	if checkErrorJSON("make_asynqreq error", err, w) {
		return
	}

	json, err := mf.FromJson([]byte(data))
	if checkErrorJSON("FromJson error parse data", err, w) {
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

	//---------------------------------------------------------------
	hash := mf.StrMd5([]byte(xml + data))
	db_path := "/" + mf.AppPath() + "/db/DB1.FDB"
	//db_path = mf.StrRegexpReplace(db_path, ":", "")
	db_path = mf.StrRegexpReplace(db_path, "[\\\\]", "/")

	db, err := sql.Open("firebirdsql", "sysdba:masterkey@127.0.0.1"+db_path)
	if checkErrorJSON("FromJson error: body "+string(body), err, w) {
		return
	}
	/***********************************************************
	IDC           VARCHAR(36),
	IDC_HUIS      VARCHAR(100),
	IN_FILE       VARCHAR(2000),
	FHASH         VARCHAR(100),
	OUT_FILE      VARCHAR(2000),
	DATE_CREATE   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	DATE_UPDATE   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	DATE_RUN      TIMESTAMP,
	DATE_END      TIMESTAMP,
	STATUS        SMALLINT     -- 0 - prepare, 1 - send, 2 - wait, 3 - end
	************************************************************/
	query := "SELECT messageguid,in_file,cdate_run,cdate_end,senderid,huisver,datareq FROM task WHERE fhash='" + hash + "'"
	rows, err := db.Query(query)
	if checkErrorJSON("db.Query error: "+query, err, w) {
		return
	}

	var messageguid, out_file, cdate_run, cdate_end, senderid, huisver, datareq string
	found := 0
	for rows.Next() {
		found++
		err = rows.Scan(&messageguid, &out_file, &cdate_run, &cdate_end, &senderid, &huisver, &datareq)
		if checkErrorJSON("rows.Scan error: "+query, err, w) {
			return
		}
		fmt.Printf("%s|%s|%s|%s|%s|%s|%s\n", messageguid, out_file, cdate_run, cdate_end, senderid, huisver, len(datareq))
	}
	if found > 0 {
		json_ret, err := get_result(messageguid, huisver, senderid, datareq, out_file)
		if checkErrorJSON("get_result error", err, w) {
			return
		}
		w.Header().Set("Content-Type", "text/json; charset=utf-8")
		w.Write(json_ret)
		return
	}
	//--------------------------------------------------------------
	{ //получаем senderid из xml:
		doc := xmlx.New()
		err = doc.LoadString(xml, nil)
		if checkErrorJSON("xmlx.LoadString error", err, w) {
			return
		}
		node := doc.SelectNode("*", "Envelope")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "Envelope") error`, errors.New("node == nil"), w)
			return
		}
		node = node.SelectNode("*", "Header")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "Header") error`, errors.New("node == nil"), w)
			return
		}
		node = doc.SelectNode("*", "RequestHeader")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "RequestHeader") error`, errors.New("node == nil"), w)
			return
		}

		senderid = node.S("*", "SenderID")
		fmt.Printf("senderid = %s\n", senderid)
	}
	datetime := mf.CurTimeStrShort()
	path := mf.AppPath() + "/temp_asyncreq/" + datetime[0:8] + "/" + datetime[9:11]
	path = strings.Replace(path, "\\", "/", -1)
	mf.MkdirAll(path)

	file := path + "/" + mf.StrRegexpReplace(signelem, "[: \\[\\]\\?\\(\\)\\&\\%\"'`]", "-") + "_" + mf.CurTimeStrShort()
	mf.FileWriteStr(file+".req_xml", xml)
	mf.FileWriteStr(file+".req_data", data)

	respbody, err := sendquery(xml, data)
	if checkErrorJSON("sendquery error", err, w) {
		return
	}

	{ //получаем messageguid из respbody
		/*********
		<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
		<soap:Envelope xmlns:ns4="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/" xmlns:ns3="http://www.w3.org/2000/09/xmldsig#" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns5="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/house-management/">
		    <soap:Header>
		        <ns4:ResultHeader>
		            <ns4:Date>2016-06-19T15:57:34.836+03:00</ns4:Date>
		            <ns4:MessageGUID>7d744f3a-02ca-4ab0-929f-e4247a033dbd</ns4:MessageGUID>
		        </ns4:ResultHeader>
		    </soap:Header>
		    <soap:Body>
		        <ns4:AckRequest>
		            <ns4:Ack>
		                <ns4:MessageGUID>         b484c8cd-55c6-4fdc-a444-97d78bc2276f</ns4:MessageGUID>
		                <ns4:RequesterMessageGUID>7d744f3a-02ca-4ab0-929f-e4247a033dbd</ns4:RequesterMessageGUID>
		            </ns4:Ack>
		        </ns4:AckRequest>
		    </soap:Body>
		</soap:Envelope>
		***********/
		doc := xmlx.New()
		err = doc.LoadString(respbody, nil)
		if checkErrorJSON("xmlx.LoadString(sendquery.respbody) error", err, w) {
			return
		}
		node := doc.SelectNode("*", "Envelope")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "Envelope") error`, errors.New("node == nil"), w)
			return
		}
		node = node.SelectNode("*", "Body")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "Body") error`, errors.New("node == nil"), w)
			return
		}
		node = doc.SelectNode("*", "AckRequest")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "AckRequest") error`, errors.New("node == nil"), w)
			return
		}
		node = doc.SelectNode("*", "Ack")
		if node == nil {
			checkErrorJSON(`doc.SelectNode("*", "Ack") error`, errors.New("node == nil"), w)
			return
		}

		//messageguid = node.S("*", "RequesterMessageGUID")
		messageguid = node.S("*", "MessageGUID")
		fmt.Printf("messageguid = %s\n", messageguid)
	}

	{ //получаем huisver из respbody
		/*********
		<soap:Envelope xmlns:ns4="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/" xmlns:ns3="http://www.w3.org/2000/09/xmldsig#" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns5="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/house-management/">
		***********/
		i := strings.Index(respbody, "<soapenv:Header>")
		if i < 0 {
			i = len(respbody) - 10
		}
		s := respbody[5:i]
		huisver = mf.RegexpCompile(`\d{1,2}\.\d+.\d+.\d+`).FindString(s)
	}

	datareq_s := string(render_data_getstate(data))
	datareq_s = strings.Replace(datareq_s, "'", "''", -1)

	_, err = db.Exec("insert into task(fhash, senderid, messageguid, huisver, in_file, cdate_run, status, datareq) " +
		"values ('" + hash + "', '" + senderid + "', '" + messageguid + "', '" + huisver + "', '" + file + "','" + datetime + "',1,'" + datareq_s + "')")
	if checkErrorJSON("db.Exec(insert into task) error", err, w) {
		return
	}

	w.Header().Set("Content-Type", "text/json; charset=utf-8")

	var ret []string
	ret = append(ret, respbody)
	ret = append(ret, messageguid)
	json_ret, _ := mf.ToJson(ret)
	if checkErrorJSON("ToJson(ret) error", err, w) {
		return
	}
	w.Write([]byte(json_ret))
}

//исправляем синхрооный запрос в асинхронный )
func make_asynqreq(xml, data string) (rxml, rdata string, err error) {
	json, err := mf.FromJson([]byte(data))
	if err != nil {
		return "", "", errors.New("ERROR make_asynqreq: FromJson(data)")
	}
	if _, ok := json["url"].(string); !ok {
		return "", "", errors.New("ERROR make_asynqreq: json param1 \"url\" not found!!")
	}
	//json["urlold"] = json["url"].(string)
	url := json["url"].(string)
	url = mf.StrRegexpReplace(url, "/$", "")
	if !mf.StrRegexpMatch(`Async$`, url) {
		json["urlsync"] = json["url"].(string)
		json["url"] = url + "Async"
	}
	bdata, err := mf.ToJson(json)
	if err != nil {
		return "", "", errors.New("ERROR make_asynqreq: ToJson(json)")
	}
	return xml, string(bdata), nil
}

//получаем данные запроса или обновляем его статус
func get_result(messageguid, huisver, senderid, datareq, file string) ([]byte, error) {
	/*********
	if file != "" {
		return get_ready_file(out_file)
	}
	*********/
	xml := render_xml_getstate(messageguid, huisver, senderid)
	data := datareq

	result, err := sendquery(xml, data)
	if err != nil {
		return nil, err
	}

	var ret []string
	ret = append(ret, result)
	ret = append(ret, data)

	json_ret, err := mf.ToJson(ret)
	if err != nil {
		return nil, err
	}

	state := check_requeststate_in_xml(result)
	if state != "2" {
		mf.FileWriteStr(file+".res_xml", result)
		mf.FileWriteStr(file+".res_data", state)
	}

	return json_ret, nil
}

//отправляем ранее сохраненные xml и data файлы в json строке
func get_ready_file(out_file string) ([]byte, error) {
	return nil, nil
}

//задаем заголовки для получения статуса запроса
func render_data_getstate(data string) []byte {
	data = strings.Trim(data, " \n\r\t")
	if data[0:1] != "{" {
		data = "{" + data + "}"
	}

	json, _ := mf.FromJson([]byte(data))
	if headers, ok := json["headers"].(map[string]interface{}); ok {
		for key, _ := range headers {
			if key == "SOAPAction" {
				headers["SOAPAction"] = "\"urn:getState\""
				break
			}
		}
		json["headers"] = headers
	}
	json_ret, _ := mf.ToJson(json)
	return json_ret
}

func check_requeststate_in_xml(xml string) string {
	/*******************************
		<soap:Envelope xmlns:ns3="http://www.w3.org/2000/09/xmldsig#" xmlns:ns4="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/" xmlns:ns5="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/house-management/" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	    <soap:Header>
	        <ns4:ResultHeader>
	            <ns4:Date>2016-06-19T17:57:04.742+03:00</ns4:Date>
	            <ns4:MessageGUID>2d36f70e-bbc7-41f4-81ca-75387a7d14b1</ns4:MessageGUID>
	        </ns4:ResultHeader>
	    </soap:Header>
	    <soap:Body>
	        <ns5:getStateResult Id="signed-data-container"><ds:Signature Id="xmldsig-95cc73da-7f21-4863-8a04-b787c56e6824" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#gostr34102001-gostr3411"/><ds:Reference URI="#signed-data-container"><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/><ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#gostr3411"/><ds:DigestValue>y4s0onGSnM9szIaPAdp4cNkO6k+tfUQ9zEsRTwE43GU=</ds:DigestValue></ds:Reference><ds:Reference Type="http://uri.etsi.org/01903#SignedProperties" URI="#xmldsig-95cc73da-7f21-4863-8a04-b787c56e6824-signedprops"><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#gostr3411"/><ds:DigestValue>QtYsyhd4b5c68XyF72HxdaF7cZ9+iLSrRSkHlWbJ7Mo=</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>aZGNDOo64yaOStdhehjQjB4okL/7gOejTVTdZYcn9SPuy9eg9sL2raRbEQuwiJvl0S4v+zAZkPNY/ywIyCfXSA==</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIDJjCCAtWgAwIBAgITEgAPldx4Rzy1qK2yugAAAA+V3DAIBgYqhQMCAgMwfzEjMCEGCSqGSIb3DQEJARYUc3VwcG9ydEBjcnlwdG9wcm8ucnUxCzAJBgNVBAYTAlJVMQ8wDQYDVQQHEwZNb3Njb3cxFzAVBgNVBAoTDkNSWVBUTy1QUk8gTExDMSEwHwYDVQQDExhDUllQVE8tUFJPIFRlc3QgQ2VudGVyIDIwHhcNMTYwNDA4MTUwMjM0WhcNMTYwNzA4MTUxMjM0WjA1MRYwFAYDVQQDDA1TSUdOXzIwMTYwNDA4MQ4wDAYDVQQKDAVMQU5JVDELMAkGA1UEBhMCUlUwYzAcBgYqhQMCAhMwEgYHKoUDAgIkAAYHKoUDAgIeAQNDAARALZ3Ofpw2FChFbnSMTGtyJiXWmC08mYp0uM/zCkY0qoegHEJnjM1EDpAzKxqwIG5ZBv0WIUOMxu58jwGibOZoUqOCAXAwggFsMA4GA1UdDwEB/wQEAwIE8DATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUkHi7x8gC+22lDM2WRQlLyObsZrAwHwYDVR0jBBgwFoAUFTF8sI0a3mbXFZxJUpcXJLkBeoMwWQYDVR0fBFIwUDBOoEygSoZIaHR0cDovL3Rlc3RjYS5jcnlwdG9wcm8ucnUvQ2VydEVucm9sbC9DUllQVE8tUFJPJTIwVGVzdCUyMENlbnRlciUyMDIuY3JsMIGpBggrBgEFBQcBAQSBnDCBmTBhBggrBgEFBQcwAoZVaHR0cDovL3Rlc3RjYS5jcnlwdG9wcm8ucnUvQ2VydEVucm9sbC90ZXN0LWNhLTIwMTRfQ1JZUFRPLVBSTyUyMFRlc3QlMjBDZW50ZXIlMjAyLmNydDA0BggrBgEFBQcwAYYoaHR0cDovL3Rlc3RjYS5jcnlwdG9wcm8ucnUvb2NzcC9vY3NwLnNyZjAIBgYqhQMCAgMDQQCWCaY+GtlxHONPpBEDlTp3/ZWfDSjDrsb2GeAR4EF7ENKquggevHcgbPxd8wYTFl2N59L3fhEWk25l8nP1O3M7</ds:X509Certificate></ds:X509Data></ds:KeyInfo><ds:Object><xades:QualifyingProperties Target="#xmldsig-95cc73da-7f21-4863-8a04-b787c56e6824" xmlns:xades="http://uri.etsi.org/01903/v1.3.2#"><xades:SignedProperties Id="xmldsig-95cc73da-7f21-4863-8a04-b787c56e6824-signedprops"><xades:SignedSignatureProperties><xades:SigningTime>2016-06-19T17:57:34.272+03:00</xades:SigningTime><xades:SigningCertificate><xades:Cert><xades:CertDigest><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#gostr3411"/><ds:DigestValue>xVlnHO/UsCPMapaQRroPok1o3cmrUyrJ6SBc6sEi78k=</ds:DigestValue></xades:CertDigest><xades:IssuerSerial><ds:X509IssuerName>cn=CRYPTO-PRO Test Center 2,o=CRYPTO-PRO LLC,l=Moscow,c=RU,1.2.840.113549.1.9.1=support@cryptopro.ru</ds:X509IssuerName><ds:X509SerialNumber>401418717008771244595411458054990910282765788</ds:X509SerialNumber></xades:IssuerSerial></xades:Cert></xades:SigningCertificate></xades:SignedSignatureProperties></xades:SignedProperties></xades:QualifyingProperties></ds:Object></ds:Signature>
	            <ns4:RequestState>2</ns4:RequestState>
	            <ns4:MessageGUID>2c03ae2a-8f24-46ad-851d-f5561083cc7a</ns4:MessageGUID>
	        </ns5:getStateResult>
	    </soap:Body>
		</soap:Envelope>
	********************************/
	doc := xmlx.New()
	err := doc.LoadString(xml, nil)
	if err != nil {
		return `ERROR: doc.LoadString(xml, nil)`
	}
	node := doc.SelectNode("*", "Envelope")
	if node == nil {
		return `ERROR: doc.SelectNode("*", "Envelope")`
	}
	node = node.SelectNode("*", "Body")
	if node == nil {
		return `ERROR: node.SelectNode("*", "Body")`
	}
	node = node.SelectNode("*", "getStateResult")
	if node == nil {
		return `ERROR: node.SelectNode("*", "getStateResult")`
	}
	state := node.S("*", "RequestState")
	return state
}

func render_xml_getstate(messageguid, huisver, senderid string) string {
	/****************************
	  <soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns="http://dom.gosuslugi.ru/schema/integration/8.7.2.2/">
	     <soapenv:Header>
	        <ns:RequestHeader>
	           <ns:Date>?</ns:Date>
	           <ns:MessageGUID>?</ns:MessageGUID>
	           <ns:SenderID>?</ns:SenderID>
	           <!--Optional:-->
	           <ns:IsOperatorSighnature>true</ns:IsOperatorSighnature>
	        </ns:RequestHeader>
	     </soapenv:Header>
	     <soapenv:Body>
	        <ns:getStateRequest>
	           <ns:MessageGUID>?</ns:MessageGUID>
	        </ns:getStateRequest>
	     </soapenv:Body>
	  </soapenv:Envelope>
	*****************************/
	s := `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns="http://dom.gosuslugi.ru/schema/integration/` + huisver + `/">
	     <soapenv:Header>
	        <ns:RequestHeader>
	           <ns:Date>` + time.Now().Format(time.RFC3339)[0:19] + `</ns:Date>
	           <ns:MessageGUID>` + mf.StrUuid() + `</ns:MessageGUID>
	           <ns:SenderID>` + senderid + `</ns:SenderID>
	           <!--Optional:-->
	           <ns:IsOperatorSighnature>true</ns:IsOperatorSighnature>
	        </ns:RequestHeader>
	     </soapenv:Header>
	     <soapenv:Body>
	        <ns:getStateRequest>
	           <ns:MessageGUID>` + messageguid + `</ns:MessageGUID>
	        </ns:getStateRequest>
	     </soapenv:Body>
	  </soapenv:Envelope>`
	return s
}

//отправляем запрос
func sendquery(xml, data string) (string, error) {
	data = strings.Trim(data, " \n\r\t")
	if data[0:1] != "{" {
		data = "{" + data + "}"
	}

	json, err := mf.FromJson([]byte(data))
	if err != nil {
		return "", err
	}

	if _, ok := json["url"].(string); !ok {
		return "", errors.New("ERROR sendquery: json param1 \"url\" not found!!")
	}

	url := json["url"].(string)
	req := gorequest.New().Post(url)

	if basicAuth, ok := json["basicAuth"].(map[string]interface{}); ok {
		fmt.Printf("has basicAuth\n")
		req = req.SetBasicAuth(basicAuth["user"].(string), basicAuth["pass"].(string))
	} else {
		fmt.Printf("no basicAuth\n")
	}

	if reqtype, ok := json["type"].(string); ok {
		req = req.Type(reqtype)
	}

	if headers, ok := json["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Set(key, value.(string))
		}
	}

	//resp, body, errs := req.Send(xml).End()
	_, body, errs := req.Send(xml).End()

	if errs != nil {
		return "", errors.New("ERROR request Send(xml): " + fmt.Sprintf("%+v", errs))
	}
	/*******
		var ret []string
		t1 := fmt.Sprintf("%+v\n=================================\n%+v", req.Header, resp.Header)
		ret = append(ret, body)
		ret = append(ret, t1)

		json_ret, err := mf.ToJson(ret)
		if err != nil {
			return nil, err
		}

		//w.Header().Set("Content-Type", "text/json; charset=utf-8")
		//w.Write(json_ret)
		return json_ret, nil
	    ********/
	return body, nil
}
