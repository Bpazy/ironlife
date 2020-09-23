package ironlife

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"ironlife/cookiejar"
	"log"
	"os"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ironlife",
		Short: "ironlife is Archery helper for DBAer",
		Long:  `自动化的工作，钢铁侠一般的生活`,
	}
	RestyClient = initRestyClient()
)

func init() {
	initViper()
}

func initViper() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户目录失败: %+v", err)
	}
	viper.AddConfigPath(userHomeDir)
	viper.SetConfigName("ironliferc")
	viper.SetConfigType("json")
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
