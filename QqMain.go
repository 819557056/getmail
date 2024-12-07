package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"oai-register/qq"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cli, _, err := qq.InitCli()
	if err != nil {
		panic(err)
	}

	// 读取邮件
	recipient, err := qq.GetUnreadEmailsForRecipient(cli, "test20241201@pki.win")

	for i := 0; i < len(recipient); i++ {
		fmt.Printf("邮件正文:%s\n", recipient[i].Text)
	}

}
