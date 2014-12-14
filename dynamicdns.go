package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var (
	email     string = "test@test.test"
	password  string = "testtest"
	domain    string = "test.test"
	subDomain string = "test"
)

func main() {
	var domainList = &domainListType{}
	err := postMsg("https://dnsapi.cn/Domain.List", url.Values{
		"login_email":    {email},
		"login_password": {password},
		"format":         {"json"},
	}, domainList)
	if err != nil {
		printError("getDomainList", err)
		os.Exit(1)
	}

	// 处理错误
	switch domainList.Status.Code {
	case "1":
		printInfo("Login", "登录成功")
	case "-1":
		printError("Login", "登录失败")
		os.Exit(1)
	}

	// 获取Domain ID
	var domainID string
	for _, v := range domainList.Domains {
		if v.Name == domain {
			domainID = strconv.Itoa(v.ID)
			break
		}
	}
	if domainID == "" {
		printError("DomainID", "账户中不存在此域名")
		printInfo("Domain", "尝试添加此域名到账户，请注意设置域名NS以及验证是否添加成功")
		var info = &infoType{}
		err = postMsg("https://dnsapi.cn/Domain.Create", url.Values{
			"login_email":    {email},
			"login_password": {password},
			"format":         {"json"},
			"domain":         {domain},
		}, info)
		if err != nil {
			printError("addDomain", err)
			os.Exit(1)
		}

		// 处理错误
		switch info.Status.Code {
		case "1":
			printInfo("addDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
		case "6":
			printError("addDomain", "域名无效")
			os.Exit(1)
		case "11":
			printError("addDomain", "域名已经存在并且是其它域名的别名")
			os.Exit(1)
		case "12":
			printError("addDomain", "域名已经存在并且您没有权限管理")
			os.Exit(1)
		case "41":
			printError("addDomain", "网站内容不符合DNSPod解析服务条款，域名添加失败")
			os.Exit(1)
		}
	}
	printInfo("DomainID", domainID)

	var recordList = &recordListType{}
	err = postMsg("https://dnsapi.cn/Record.List", url.Values{
		"login_email":    {email},
		"login_password": {password},
		"format":         {"json"},
		"domain_id":      {domainID},
	}, recordList)
	if err != nil {
		printError("getRecordList", err)
		os.Exit(1)
	}

	// 获取Record ID
	var recordID string
	for _, v := range recordList.Records {
		// 这里只根据子域名相同和类型为A记录
		// 获取第一个匹配的ID
		if v.Name == subDomain && v.Type == "A" {
			recordID = v.ID
			break
		}
	}
	if recordID == "" {
		printError("RecordID", "域名中不存在此子域名或此子域名不存在A记录")
		printInfo("SubDomain", "尝试添加此子域名A记录")
		var info = &infoType{}
		err = postMsg("https://dnsapi.cn/Record.Create", url.Values{
			"login_email":    {email},
			"login_password": {password},
			"format":         {"json"},
			"domain_id":      {domainID},
			"sub_domain":     {subDomain},
			"record_type":    {"A"},
			"record_line":    {"默认"},
			"value":          {"21.21.21.21"},
		}, info)
		if err != nil {
			printError("addSubDomain", err)
			os.Exit(1)
		}

		// 处理错误
		switch info.Status.Code {
		case "1":
			printInfo("addSubDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
		case "-15":
			printError("addSubDomain", "域名已被封禁")
			os.Exit(1)
		case "-7":
			printError("addSubDomain", "企业账号的域名需要升级才能设置")
			os.Exit(1)
		case "-8":
			printError("addSubDomain", "代理名下用户的域名需要升级才能设置")
			os.Exit(1)
		case "21":
			printError("addSubDomain", "域名被锁定")
			os.Exit(1)
		case "22":
			printError("addSubDomain", "子域名不合法")
			os.Exit(1)
		case "23":
			printError("addSubDomain", "子域名级数超出限制")
			os.Exit(1)
		case "24":
			printError("addSubDomain", "泛解析子域名错误")
			os.Exit(1)
		case "25":
			printError("addSubDomain", "轮循记录数量超出限制")
			os.Exit(1)
		case "31":
			printError("addSubDomain", "存在冲突的记录(A记录、CNAME记录、URL记录不能共存)")
			os.Exit(1)
		case "33":
			printError("addSubDomain", "AAAA 记录数超出限制")
			os.Exit(1)
		case "82":
			printError("addSubDomain", "不能添加黑名单中的IP")
			os.Exit(1)
		}
	}
	printInfo("RecordID", recordID)

	// 设置动态DNS
	var recordModify = &infoType{}
	err = postMsg("https://dnsapi.cn/Record.Ddns", url.Values{
		"login_email":    {email},
		"login_password": {password},
		"format":         {"json"},
		"domain_id":      {domainID},
		"record_id":      {recordID},
		"sub_domain":     {subDomain},
		"record_line":    {"默认"},
	}, recordModify)
	if err != nil {
		printError("getRecordList", err)
		os.Exit(1)
	}

	printInfo("log", recordModify)
}

func postMsg(u string, msg url.Values, value interface{}) error {
	getDomainList, err := http.PostForm(u, msg)
	if err != nil {
		return err
	}
	defer getDomainList.Body.Close()

	buf, err := ioutil.ReadAll(getDomainList.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, value)
	if err != nil {
		return err
	}

	return nil
}

func printInfo(s string, v ...interface{}) {
	log.Println(append([]interface{}{"[INFO]", s + ":"}, v...)...)
}

func printError(s string, v ...interface{}) {
	log.Println(append([]interface{}{"[ERROR]", s + ":"}, v...)...)
}

type domainListType struct {
	Status  statusType   `json:"status"`
	Info    struct{}     `json:"info"`
	Domains []domainType `json:"domains"`
}

type recordListType struct {
	Status  statusType   `json:"status"`
	Info    struct{}     `json:"info"`
	Domain  domainType   `json:"domain"`
	Records []recordType `json:"records"`
}

type infoType struct {
	Status statusType `json:"status"`
	Record recordType `json:"record"`
}

type statusType struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type domainType struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Grade            string `json:"grade"`
	GradeTitle       string `json:"grade_title"`
	ExtStatus        string `json:"ext_status"`
	Records          string `json:"records"`
	GroupID          string `json:"group_id"`
	IsMark           string `json:"is_mark"`
	Remark           string `json:"remark"`
	IsVIP            string `json:"is_vip"`
	SearchenginePush string `json:"searchengine_push"`
	Beian            string `json:"beian"`
	CreatedOn        string `json:"created_on"`
	UpdatedOn        string `json:"updated_on"`
	TTL              string `json:"ttl"`
	Owner            string `json:"owner"`
}

type recordType struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Line          string `json:"line"`
	Type          string `json:"type"`
	TTL           string `json:"ttl"`
	Value         string `json:"value"`
	MX            string `json:"mx"`
	Enabled       string `json:"enabled"`
	Status        string `json:"status"`
	MonitorStatus string `json:"monitor_status"`
	Remark        string `json:"remark"`
	UpdatedOn     string `json:"updated_on"`
	Hold          string `json:"hold"`
}
