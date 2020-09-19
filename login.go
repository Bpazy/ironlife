package ironlife

import (
	"encoding/json"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
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
			username, password := PromptUsernamePassword2()
			success, errMsg := Login(username, password)
			if success {
				viper.Set(UsernameConfigKey, username)
				viper.Set(PasswordConfigKey, password)
				err := viper.SafeWriteConfig()
				if err != nil {
					log.Fatalf("保存配置文件失败: %+v", err)
				}
				break
			}
			fmt.Printf("Login failed. Please reinput username password. Fail message: %s\n", errMsg)
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

func PromptUsernamePassword2() (username, password string) {
	fmt.Print("Archery username? ")
	fmt.Scanln(&username)
	fmt.Print("Archery password? ")
	fmt.Scanln(&password)
	return
}

type LoginResult struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   string `json:"data"`
}

// Login return login status and error message
func Login(username string, password string) (bool, string) {
	// 获取基础 Cookie
	_, _ = RestyClient.R().Get(GetBaseUrl() + "/login/")
	// 登录
	loginRsp, err := RestyClient.R().
		SetFormData(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(GetBaseUrl() + "/authenticate/")
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

func GetBaseUrl() string {
	return viper.GetString(BaseUrlConfigKey)
}
