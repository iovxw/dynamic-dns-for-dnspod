package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	ipRegexp = regexp.MustCompile(`<code>([^\n]+)</code>`)

	email     string        = "test@test.test"
	password  string        = "testtest"
	domain    string        = "test.test"
	subDomain string        = "test"
	checkTime time.Duration = 30
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
	if isError(domainList.Status.Code) {
		os.Exit(1)
	} else {
		printInfo("Login", "登录成功")
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
		if isError(info.Status.Code) {
			os.Exit(1)
		} else {
			printInfo("addDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
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
		// 获取相同子域名，类型为A记录，线路为默认的记录ID
		if v.Name == subDomain && v.Type == "A" && v.Line == "默认" {
			recordID = v.ID
			break
		}
	}
	if recordID == "" {
		printError("RecordID", "域名中不存在此子域名或此子域名不存在默认线路的A记录")
		printInfo("SubDomain", "尝试添加此子域名A记录")
		// 获取当前IP
		ip, err := getIP()
		if err != nil {
			printError("getIP", err)
		}
		var info = &infoType{}
		err = postMsg("https://dnsapi.cn/Record.Create", url.Values{
			"login_email":    {email},
			"login_password": {password},
			"format":         {"json"},
			"domain_id":      {domainID},
			"sub_domain":     {subDomain},
			"record_type":    {"A"},
			"record_line":    {"默认"},
			"value":          {ip},
		}, info)
		if err != nil {
			printError("addSubDomain", err)
			os.Exit(1)
		}

		// 处理错误
		if isError(info.Status.Code) {
			os.Exit(1)
		} else {
			printInfo("addSubDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
		}
	}
	printInfo("RecordID", recordID)

	for {
		// 获取记录的IP
		var recordInfo = &recordInfoType{}
		err = postMsg("https://dnsapi.cn/Record.Info", url.Values{
			"login_email":    {email},
			"login_password": {password},
			"format":         {"json"},
			"domain_id":      {domainID},
			"record_id":      {recordID},
		}, recordInfo)
		if err != nil {
			printError("getRecordInfo", err)
		} else {
			// 处理错误
			if !isError(recordInfo.Status.Code) {
				// 无错误

				// 获取当前IP
				ip, err := getIP()
				if err != nil {
					printError("getIP", err)
				} else {
					// 检查DNS记录的IP是否和当前IP相同
					if ip != recordInfo.Record.Value {
						printInfo("IP", recordInfo.Record.Value, "==>", ip, "IP变更，自动更新DNS")
						// 设置动态DNS
						var recordModify = &recordModifyType{}
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
							printError("recordModify", err)
						} else {
							// 处理错误
							if !isError(recordModify.Status.Code) {
								// 无错误

								printInfo("IP", "DNS更新完成")
							}
						}
					} else {
						printInfo("IP", "没有检测到IP变更。下次检查时间：", checkTime, "后")
					}
				}
			}
		}

		time.Sleep(time.Second * checkTime)
	}
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

func isError(code string) bool {
	switch code {
	case "1":
		return false
	case "-1":
		printError("Login", "登录失败")
	case "-2":
		printError("Login", "API使用超出限制")
	case "-8":
		printError("Login", "登录失败次数过多，帐号被暂时封禁")
	case "83":
		printError("Login", "该帐户已经被锁定，无法进行任何操作")
	case "85":
		printError("Login", "该帐户开启了登录区域保护，当前IP不在允许的区域内")
	case "6":
		printError("addDomain", "域名无效")
	case "11":
		printError("addDomain", "域名已经存在并且是其它域名的别名")
	case "12":
		printError("addDomain", "域名已经存在并且您没有权限管理")
	case "41":
		printError("addDomain", "网站内容不符合DNSPod解析服务条款，域名添加失败")
	case "-15":
		printError("addSubDomain", "域名已被封禁")
	case "-7":
		printError("addSubDomain", "企业账号的域名需要升级才能设置")
	case "21":
		printError("addSubDomain", "域名被锁定")
	case "22":
		printError("addSubDomain", "子域名不合法")
	case "23":
		printError("addSubDomain", "子域名级数超出限制")
	case "24":
		printError("addSubDomain", "泛解析子域名错误")
	case "25":
		printError("addSubDomain", "轮循记录数量超出限制")
	case "31":
		printError("addSubDomain", "存在冲突的记录(A记录、CNAME记录、URL记录不能共存)")
	case "33":
		printError("addSubDomain", "AAAA 记录数超出限制")
	case "82":
		printError("addSubDomain", "不能添加黑名单中的IP")
	default:
		printError("DNSPodAPI", "未知错误，错误代号:", code)
	}
	return true
}

func getIP() (string, error) {
	resp, err := http.Get("http://ip.cn")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 提取IP地址
	buf := ipRegexp.FindSubmatch(body)
	// 页面内不存在IP地址
	if len(buf) == 0 {
		return "", errors.New("服务器错误，无法获取当前ip")
	}

	return string(buf[1]), nil
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

type recordInfoType struct {
	Status statusType `json:"status"`
	Domain domainType `json:"domain"`
	Record recordType `json:"record"`
}

// 这里单独一个而不用上面那个infoType的原因是
// DNSPod的动态DNS API里返回的Record ID
// 竟然和别的API里的不一样
// 是int而不是string
type recordModifyType struct {
	Status statusType `json:"status"`
	Record struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"record"`
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
