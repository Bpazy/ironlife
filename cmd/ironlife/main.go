package main

import (
	"encoding/json"
	"github.com/c-bata/go-prompt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
	cookiejar "ironlife"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	usernameConfigKey = "username"
	passwordConfigKey = "password"
	baseUrlConfigKey  = "baseUrl"
)

var (
	debug = false
)

func init() {
	debug = *kingpin.Flag("debug", "Enable Debug mod").Bool()
}

func main() {
	username, password := initConfig()

	client := initRestyClient()
	login(client, username, password)

	log.Println("初始化完毕，开始 30s 轮训")
	for {
		queryAndApprove(client)
		time.Sleep(30 * time.Second)
	}
}

func queryAndApprove(client *resty.Client) {
	// 查询权限列表
	queryListRsp, err := client.R().
		SetFormData(map[string]string{
			"limit":  "20",
			"offset": "0",
			"search": "",
		}).
		Post(getBaseUrl() + "/query/applylist/")
	if err != nil {
		log.Fatalf("查询权限管理列表失败：%+v", err)
	}
	if queryListRsp.StatusCode() != http.StatusOK {
		log.Fatalf("查询权限管理列表失败，返回码：%d，返回值：%s", queryListRsp.StatusCode(), queryListRsp.String())
	}
	applyListResult := ApplyListResult{}
	err = json.Unmarshal(queryListRsp.Body(), &applyListResult)
	if err != nil {
		if debug {
			log.Println("username: " + viper.GetString(usernameConfigKey))
			log.Println("password: " + viper.GetString(passwordConfigKey))
			log.Println("baseUrl: " + viper.GetString(baseUrlConfigKey))
		}
		log.Fatalf("权限管理列表接口返回值转为 Json 失败，请检查用户名密码是否正确: %+v, body: \n%s", err, queryListRsp.String())
	}

	csrfReg := regexp.MustCompile("<input type=\"hidden\" name=\"csrfmiddlewaretoken\" value=\"(.+)\">")
	for _, row := range applyListResult.Rows {
		validDate, err := time.Parse("2006-01-02", row.ValidDate)
		if err != nil {
			log.Printf("parse valid_date 失败，跳过当前工单, err: %+v\n", err)
			continue
		}
		isApplyLessThan7Day := validDate.Sub(time.Now()).Hours()/24 < 7
		if row.LimitNum <= 100 && isApplyLessThan7Day && row.Status == 0 {
			applyId := strconv.Itoa(row.ApplyID)
			// 通过审批
			csrfRsp, err := client.R().Get(getBaseUrl() + "/queryapplydetail/" + applyId)
			if err != nil {
				log.Printf("获取审批详情页中的csrfmiddlewaretoken失败，跳过此工单，err: %+v\n", err)
				continue
			}
			submatch := csrfReg.FindStringSubmatch(csrfRsp.String())
			if len(submatch) < 1 {
				log.Println("获取审批详情页中的csrfmiddlewaretoken失败，工单可能已被审批，跳过此工单")
				continue
			}
			matchedCsrfToken := submatch[1]
			_, _ = client.R().
				SetFormData(map[string]string{
					"csrfmiddlewaretoken": matchedCsrfToken,
					"apply_id":            applyId,
					"audit_status":        "1", // 通过是1，终止是2
				}).
				Post(getBaseUrl() + "/query/privaudit/")
			log.Printf("%s(%s) 审批通过\n", row.Title, row.UserDisplay)
		}
	}
}

func login(client *resty.Client, username string, password string) {
	// 获取基础 Cookie
	_, _ = client.R().Get(getBaseUrl() + "/login/")
	// 登录
	loginRsp, err := client.R().
		SetFormData(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(getBaseUrl() + "/authenticate/")
	if err != nil {
		log.Fatalf("登录失败: %+v", err)
	}
	if loginRsp.StatusCode() != http.StatusOK {
		log.Fatalf("登录失败，返回码：%d，返回值：%s", loginRsp.StatusCode(), loginRsp.String())
	}
	loginResult := LoginResult{}
	err = json.Unmarshal(loginRsp.Body(), &loginResult)
	if err != nil {
		log.Fatalf("登录失败，序列化接口返回值失败: %+v", err)
	}
	if loginResult.Status != 0 {
		log.Fatalf("登录失败，用户名或密码错误")
	}
}

func initRestyClient() *resty.Client {
	client := resty.New()
	jar, err := cookiejar.NewCookieJar(client)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	client.SetCookieJar(jar)
	return client
}

func initConfig() (string, string) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户目录失败: %+v", err)
	}
	viper.AddConfigPath(userHomeDir)
	viper.SetConfigName("ironliferc")
	viper.SetConfigType("json")
	viper.SetDefault(baseUrlConfigKey, "http://archery.example.com")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			promptUsernamePassword()
			err = viper.SafeWriteConfig()
			if err != nil {
				log.Fatalf("保存配置文件失败: %+v", err)
			}
		} else {
			// Config file was found but another error was produced
			log.Fatalf("读取配置文件失败: %+v", err)
		}
	}

	username := viper.GetString(usernameConfigKey)
	password := viper.GetString(passwordConfigKey)
	for username == "" || password == "" {
		username, password = promptUsernamePassword()
	}
	err = viper.WriteConfig()
	if err != nil {
		log.Fatalf("保存配置文件失败: %+v", err)
	}
	return username, password
}

type ApplyListResult struct {
	Total int `json:"total"`
	Rows  []struct {
		ApplyID              int    `json:"apply_id"`
		Title                string `json:"title"`
		InstanceInstanceName string `json:"instance__instance_name"`
		DbList               string `json:"db_list"`
		PrivType             int    `json:"priv_type"`
		TableList            string `json:"table_list"`
		LimitNum             int    `json:"limit_num"`
		ValidDate            string `json:"valid_date"`
		UserDisplay          string `json:"user_display"`
		Status               int    `json:"status"`
		CreateTime           string `json:"create_time"`
		GroupName            string `json:"group_name"`
	} `json:"rows"`
}

type LoginResult struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   string `json:"data"`
}

func promptUsernamePassword() (username, password string) {
	emptyCompleter := func(document prompt.Document) []prompt.Suggest {
		return nil
	}
	username = prompt.Input("Archery username? ", emptyCompleter)
	password = prompt.Input("Archery password? ", emptyCompleter)

	viper.Set(usernameConfigKey, username)
	viper.Set(passwordConfigKey, password)
	return
}

func getBaseUrl() string {
	return viper.GetString(baseUrlConfigKey)
}
