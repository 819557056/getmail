package main

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 连接到 QQ 邮箱的 IMAP 服务器
	c, err := client.DialTLS("imap.qq.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Logout()

	qqMail := os.Getenv("QQ_MAIL")
	qqPwd := os.Getenv("QQ_PWD")

	// 登录
	if err := c.Login(qqMail, qqPwd); err != nil {
		log.Fatal(err)
	}

	// 选择收件箱
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// 获取最新的10封邮件
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	// 获取邮件的基本信息
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	// 打印邮件信息
	for msg := range messages {
		env := msg.Envelope
		from := ""
		if len(env.From) > 0 {
			addr := env.From[0]
			if addr.PersonalName != "" {
				from = addr.PersonalName
			} else {
				from = addr.MailboxName + "@" + addr.HostName
			}
		}
		fmt.Printf("主题: %v\n", env.Subject)
		fmt.Printf("发件人: %v\n", from)
		fmt.Printf("日期: %v\n", env.Date)
		fmt.Printf("------------------------\n")
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}
}
