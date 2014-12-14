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

	"gopkg.in/v2/yaml"
)

var (
	// 用于截取当前IP，放在全局变量是为了不用多次解析正则
	ipRegexp = regexp.MustCompile(`<code>([^\n]+)</code>`)
)

func main() {
	// 读配置文件
	buf, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		printError("readConfig", err)
		os.Exit(1)
	}
	var config = new(configType)
	err = yaml.Unmarshal(buf, config)
	if err != nil {
		printError("unmarshalConfig", err)
		os.Exit(1)
	}

	var domainList = new(domainListType)
	err = postMsg("https://dnsapi.cn/Domain.List", url.Values{
		"login_email":    {config.Email},
		"login_password": {config.Password},
		"format":         {"json"},
		"lang":           {"cn"},
	}, domainList)
	if err != nil {
		printError("getDomainList", err)
		os.Exit(1)
	}

	// 检查错误
	if domainList.Status.Code == "1" {
		printInfo("Login", "登录成功")
	} else {
		printError("Login", domainList.Status.Message)
		os.Exit(1)
	}

	// 获取Domain ID
	var domainID string
	for _, v := range domainList.Domains {
		if v.Name == config.Domain {
			domainID = strconv.Itoa(v.ID)
			break
		}
	}
	if domainID == "" {
		printError("DomainID", "账户中不存在此域名")
		printInfo("Domain", "尝试添加此域名到账户，请注意设置域名NS以及验证是否添加成功")
		var info = new(infoType)
		err = postMsg("https://dnsapi.cn/Domain.Create", url.Values{
			"login_email":    {config.Email},
			"login_password": {config.Password},
			"format":         {"json"},
			"lang":           {"cn"},
			"domain":         {config.Domain},
		}, info)
		if err != nil {
			printError("addDomain", err)
			os.Exit(1)
		}

		// 检查错误
		if info.Status.Code == "1" {
			printInfo("addDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
		} else {
			printError("addDomain", info.Status.Message)
			os.Exit(1)
		}
	}
	printInfo("DomainID", domainID)

	var recordList = new(recordListType)
	err = postMsg("https://dnsapi.cn/Record.List", url.Values{
		"login_email":    {config.Email},
		"login_password": {config.Password},
		"format":         {"json"},
		"lang":           {"cn"},
		"domain_id":      {domainID},
	}, recordList)
	if err != nil {
		printError("getRecordList", err)
		os.Exit(1)
	}

	// 检查错误
	if recordList.Status.Code != "1" {
		printError("getRecordList", recordList.Status.Message)
		os.Exit(1)
	}

	// 获取Record ID
	var recordID string
	for _, v := range recordList.Records {
		// 获取相同子域名，类型为A记录，线路为默认的记录ID
		if v.Name == config.SubDomain && v.Type == "A" && v.Line == "默认" {
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
		var info = new(infoType)
		err = postMsg("https://dnsapi.cn/Record.Create", url.Values{
			"login_email":    {config.Email},
			"login_password": {config.Password},
			"format":         {"json"},
			"lang":           {"cn"},
			"domain_id":      {domainID},
			"sub_domain":     {config.SubDomain},
			"record_type":    {"A"},
			"record_line":    {"默认"},
			"value":          {ip},
		}, info)
		if err != nil {
			printError("addSubDomain", err)
			os.Exit(1)
		}

		// 检查错误
		if info.Status.Code == "1" {
			printInfo("addSubDomain", "操作成功，请重启程序查看是否可用")
			// 操作成功，退出程序
			os.Exit(0)
		} else {
			printError("addSubDomain", info.Status.Message)
			os.Exit(1)
		}
	}
	printInfo("RecordID", recordID)

	// 循环检测IP与设置DNS
	// 一旦出现error则直接结束本次循环
	// 进入sleep等待下一次循环
	for {
		// 获取记录的IP
		var recordInfo = new(recordInfoType)
		err = postMsg("https://dnsapi.cn/Record.Info", url.Values{
			"login_email":    {config.Email},
			"login_password": {config.Password},
			"format":         {"json"},
			"lang":           {"cn"},
			"domain_id":      {domainID},
			"record_id":      {recordID},
		}, recordInfo)
		if err != nil {
			printError("getRecordInfo", err)
		} else {
			// 处理错误
			if recordInfo.Status.Code != "1" {
				printError("getRecordInfo", recordInfo.Status.Message)
			} else {
				// 获取当前IP
				ip, err := getIP()
				if err != nil {
					printError("getIP", err)
				} else {
					// 检查DNS记录的IP是否和当前IP相同
					if ip != recordInfo.Record.Value {
						printInfo("IP", recordInfo.Record.Value, "==>", ip, "IP变更，自动更新DNS")
						// 设置动态DNS
						var recordModify = new(recordModifyType)
						err = postMsg("https://dnsapi.cn/Record.Ddns", url.Values{
							"login_email":    {config.Email},
							"login_password": {config.Password},
							"format":         {"json"},
							"lang":           {"cn"},
							"domain_id":      {domainID},
							"record_id":      {recordID},
							"sub_domain":     {config.SubDomain},
							"record_line":    {"默认"},
						}, recordModify)
						if err != nil {
							printError("recordModify", err)
						} else {
							// 处理错误
							if recordModify.Status.Code == "1" {
								printInfo("IP", "DNS更新完成")
							} else {
								printError("recordModify", recordModify.Status.Message)
							}
						}
					} else {
						printInfo("IP", "没有检测到IP变更。下次检查时间：", config.CheckTime, "后")
					}
				}
			}
		}

		time.Sleep(time.Second * config.CheckTime)
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

type configType struct {
	Email     string        `yaml:"Email"`
	Password  string        `yaml:"Password"`
	Domain    string        `yaml:"Domain"`
	SubDomain string        `yaml:"SubDomain"`
	CheckTime time.Duration `yaml:"CheckTime"`
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
