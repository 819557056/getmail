package main

import (
	"fmt"
	"getmail/Cf"
	"getmail/Qq"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func getQqMail() {
	_, cli, err := Qq.InitCli()
	if err != nil {
		panic(err)
	}
	defer cli.Logout()

	// 读取邮件
	recipient, err := Qq.GetUnreadEmailsForRecipient(cli, "test20241201@pkica.win")
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(recipient); i++ {
		fmt.Printf("邮件正文:%s\n", recipient[i].Text)
	}
}

func getCfMailList() {
	//转发的目标邮箱，接收邮箱
	toMail := os.Getenv("QQ_MAIL")

	cli, container, err := Cf.InitCli()
	rules, _, err := Cf.ListTempMail(container, cli)
	if err != nil {
		panic(err)
	}
	// 打印结果
	fmt.Printf("找到 %d 条规则:\n", len(rules))
	for i, rule := range rules {
		// 检查是否有动作值为 "2498073395@qq.com"
		hasTargetAction := false
		for _, action := range rule.Actions {
			for i := 0; i < len(action.Value); i++ {
				if action.Value[i] == toMail {
					hasTargetAction = true
					break
				}
			}

		}

		if hasTargetAction {
			fmt.Printf("\n规则 #%d:\n", i+1)
			fmt.Printf("ID: %s\n", rule.Tag)
			fmt.Printf("名称: %s\n", rule.Name)
			fmt.Printf("启用状态: %v\n", rule.Enabled)

			// 打印匹配器信息
			if rule.Matchers != nil && len(rule.Matchers) > 0 {
				fmt.Printf("匹配器:\n")
				for _, matcher := range rule.Matchers {
					fmt.Printf("- 类型: %s\n", matcher.Type)
					fmt.Printf("    字段: %s\n", matcher.Field)
					fmt.Printf("    值: %v\n", matcher.Value)
				}
			}

			// 打印动作信息
			if rule.Actions != nil && len(rule.Actions) > 0 {
				fmt.Printf("动作:\n")
				for _, action := range rule.Actions {
					fmt.Printf("  - 类型: %s\n", action.Type)
					if action.Value != nil {
						fmt.Printf("    值: %v\n", action.Value)
					}
				}
			}
		}
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	getQqMail()

	//html := Qq.ExtractTextFromHTML("<div style=\"font-family: -apple-system, system-ui; font-size: 11pt; color: rgb(0, 0, 0);\"><span style=\"line-height: 1.6;\">456</span></div>")
	//fmt.Println(html)
}
