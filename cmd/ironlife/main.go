package main

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
	"ironlife"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	debug = false
)

func init() {
	debug = *kingpin.Flag("debug", "Enable Debug mod").Bool()
}

func main() {
	username, password := initConfig()

	ironlife.LoginWithConfiguredBaseUrl(username, password)

	log.Println("初始化完毕，开始 30s 轮训")
	for {
		queryAndApprove(ironlife.RestyClient)
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
		Post(ironlife.GetBaseUrl() + "/query/applylist/")
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
			log.Println("username: " + viper.GetString(ironlife.UsernameConfigKey))
			log.Println("password: " + viper.GetString(ironlife.PasswordConfigKey))
			log.Println("baseUrl: " + viper.GetString(ironlife.BaseUrlConfigKey))
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
			csrfRsp, err := client.R().Get(ironlife.GetBaseUrl() + "/queryapplydetail/" + applyId)
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
				Post(ironlife.GetBaseUrl() + "/query/privaudit/")
			log.Printf("%s(%s) 审批通过\n", row.Title, row.UserDisplay)
		}
	}
}

func initConfig() (string, string) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户目录失败: %+v", err)
	}
	viper.AddConfigPath(userHomeDir)
	viper.SetConfigName("ironliferc")
	viper.SetConfigType("json")
	viper.SetDefault(ironlife.BaseUrlConfigKey, "http://archery.example.com")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			ironlife.PromptUsernamePassword()
			err = viper.SafeWriteConfig()
			if err != nil {
				log.Fatalf("保存配置文件失败: %+v", err)
			}
		} else {
			// Config file was found but another error was produced
			log.Fatalf("读取配置文件失败: %+v", err)
		}
	}

	username := viper.GetString(ironlife.UsernameConfigKey)
	password := viper.GetString(ironlife.PasswordConfigKey)
	for username == "" || password == "" {
		username, password = ironlife.PromptUsernamePassword()
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
