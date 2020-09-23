package ironlife

import (
	"encoding/json"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strings"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login Archery",
	Long:  `Login Archery. Set your username and password`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			username := Prompt("Archery username? ")
			password := Prompt("Archery password? ")

			defaultBaseUrl := viper.GetString(BaseUrlConfigKey)
			if defaultBaseUrl == "" {
				defaultBaseUrl = "http://archery.example.com"
			}
			input := Prompt(fmt.Sprintf("Archery base url(default %s)? ", defaultBaseUrl))
			if input != "" {
				defaultBaseUrl = input
			}

			defaultBaseUrl = strings.TrimRight(CompleteProtocol(defaultBaseUrl), "/")
			success, errMsg := Login(username, password, defaultBaseUrl)
			if success {
				viper.Set(UsernameConfigKey, username)
				viper.Set(PasswordConfigKey, password)
				viper.Set(BaseUrlConfigKey, defaultBaseUrl)
				err := viper.SafeWriteConfig()
				if err != nil {
					log.Fatalf("保存配置文件失败: %+v", err)
				}
				fmt.Println("Login success")
				break
			}
			fmt.Printf("Login failed. Please check your input. Fail message: %s\n", errMsg)
		}

	},
}

const (
	UsernameConfigKey = "username"
	PasswordConfigKey = "password"
	BaseUrlConfigKey  = "baseUrl"
)

// Deprecated
func PromptUsernamePassword() (username, password string) {
	emptyCompleter := func(document prompt.Document) []prompt.Suggest {
		return nil
	}
	username = prompt.Input("Archery username? ", emptyCompleter)
	password = prompt.Input("Archery password? ", emptyCompleter)

	viper.Set(UsernameConfigKey, username)
	viper.Set(PasswordConfigKey, password)
	return
}

func Prompt(prompt string) (input string) {
	fmt.Print(prompt)
	fmt.Scanln(&input)
	return
}

type LoginResult struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   string `json:"data"`
}

func Login(username, password, baseurl string) (bool, string) {
	// 获取基础 Cookie
	_, _ = RestyClient.R().Get(baseurl + "/login/")
	// 登录
	loginRsp, err := RestyClient.R().
		SetFormData(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(baseurl + "/authenticate/")
	if err != nil {
		return false, fmt.Sprintf("登录失败: %+v", err)
	}
	if loginRsp.StatusCode() != http.StatusOK {
		return false, fmt.Sprintf("登录失败，返回码：%d，返回值：%s", loginRsp.StatusCode(), loginRsp.String())
	}
	loginResult := LoginResult{}
	err = json.Unmarshal(loginRsp.Body(), &loginResult)
	if err != nil {
		return false, fmt.Sprintf("登录失败，序列化接口返回值失败: %+v", err)
	}
	if loginResult.Status != 0 {
		return false, "登录失败，用户名或密码错误"
	}
	return true, ""
}

// LoginWithConfiguredBaseUrl return login status and error message
func LoginWithConfiguredBaseUrl(username string, password string) (bool, string) {
	return Login(username, password, GetBaseUrl())
}

func GetBaseUrl() string {
	return viper.GetString(BaseUrlConfigKey)
}
