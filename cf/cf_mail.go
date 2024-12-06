package cf

import (
	"context"
	"github.com/cloudflare/cloudflare-go"
	"log"
	"os"
)

func InitCli() (*cloudflare.API, cloudflare.ResourceContainer, error) {
	// 设置 Cloudflare API 令牌
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	//if apiToken == "" {
	//	log.Fatal("请设置 CLOUDFLARE_API_TOKEN 环境变量")
	//}

	mail := os.Getenv("CLOUDFLARE_MAIL")

	// 设置区域 ID（从你的 Cloudflare 账户获取）
	zoneID := os.Getenv("CLOUDFLARE_ZONE_ID")

	// 创建 Cloudflare API 客户端
	//api, err := cloudflare.NewWithAPIToken(apiToken)
	api, err := cloudflare.New(apiToken, mail)
	if err != nil {
		log.Fatalf("创建 API 客户端失败: %v", err)
	}

	// 创建ResourceContainer
	rc := cloudflare.ResourceContainer{
		Identifier: zoneID, Level: cloudflare.ZoneRouteLevel, // 使用区域级别
	}

	return api, rc, err
}

func CreateTempMail(rc cloudflare.ResourceContainer, api *cloudflare.API, tempMail string) error {
	//转发的目标邮箱，接收邮箱
	toMail := os.Getenv("QQ_MAIL")

	enabled := true
	// 创建规则参数
	paramsCreate := cloudflare.CreateEmailRoutingRuleParameters{
		Name:    "My Email Rule",
		Enabled: &enabled, // 设置匹配条件
		Matchers: []cloudflare.EmailRoutingRuleMatcher{
			{
				Type:  "literal",
				Field: "to",
				Value: tempMail,
			},
		}, // 设置动作
		Actions: []cloudflare.EmailRoutingRuleAction{
			{
				Type:  "forward",
				Value: []string{toMail},
			},
		},
	}

	// 调用 API创建规则
	_, err := api.CreateEmailRoutingRule(context.Background(), &rc, paramsCreate)
	if err != nil {
		log.Fatalf("创建邮件路由规则失败: %v", err)
	}

	return err
}

func ListTempMail(rc cloudflare.ResourceContainer, api *cloudflare.API) ([]cloudflare.EmailRoutingRule, *cloudflare.ResultInfo, error) {

	enabled := true
	// 创建查询参数
	params := cloudflare.ListEmailRoutingRulesParameters{
		Enabled: &enabled,
		ResultInfo: cloudflare.ResultInfo{
			Page:    1,  // 页码
			PerPage: 50, // 每页数量
		},
	}

	// 调用 API 获取邮件路由规则
	rules, resultInfo, err := api.ListEmailRoutingRules(context.Background(), &rc, params)
	if err != nil {
		log.Fatalf("获取邮件路由规则失败: %v", err)
	}

	/*// 打印结果
	fmt.Printf("找到 %d 条规则:\n", len(rules))
	for i, rule := range rules {
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

	// 打印分页信息
	if resultInfo != nil {
		fmt.Printf("\n分页信息:\n")
		fmt.Printf("总数: %d\n", resultInfo.Total)
		fmt.Printf("当前页: %d\n", resultInfo.Page)
		fmt.Printf("每页数量: %d\n", resultInfo.PerPage)
		fmt.Printf("本页数量: %d\n", resultInfo.Count)
	}*/
	return rules, resultInfo, err
}
