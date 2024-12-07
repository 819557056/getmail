package qq

import (
	"bytes"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"strings"
)

type EmailContent struct {
	Subject string
	From    string
	To      string
	Date    string
	Text    string
	HTML    string
}

// 解码邮件主题或发件人等信息
func decodeMailString(s string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

func InitCli() (*imap.MailboxStatus, *client.Client, error) {
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

	return mbox, c, err
}

func GetListMail(mbox *imap.MailboxStatus, cli *client.Client, count int) chan *imap.Message {
	// 获取最新的10封邮件
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	// 获取邮件的基本信息
	messages := make(chan *imap.Message, count)
	done := make(chan error, 1)
	go func() {
		done <- cli.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	return messages
}

func GetUnreadEmailsForRecipient(c *client.Client, recipientEmail string) ([]*EmailContent, error) {
	// 选择收件箱
	_, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("选择收件箱失败: %v", err)
	}

	// 搜索未读邮件
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	criteria.Header.Add("To", recipientEmail)

	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("搜索邮件失败: %v", err)
	}

	if len(uids) == 0 {
		return nil, nil // 没有未读邮件
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	// 获取邮件内容
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var emails []*EmailContent

	for msg := range messages {
		email, err := processEmail(msg, section)
		if err != nil {
			log.Printf("处理邮件失败: %v", err)
			continue
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("获取邮件失败: %v", err)
	}

	return emails, nil
}

func processEmail(msg *imap.Message, section *imap.BodySectionName) (*EmailContent, error) {
	r := msg.GetBody(section)
	if r == nil {
		return nil, fmt.Errorf("服务器未返回邮件内容")
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, err
	}

	header := mr.Header
	subject, _ := header.Subject()
	from, _ := header.AddressList("From")
	to, _ := header.AddressList("To")
	date, _ := header.Date()

	email := &EmailContent{
		Subject: decodeHeader(subject),
		From:    decodeHeader(from[0].String()),
		To:      decodeHeader(to[0].String()),
		Date:    date.String(),
	}

	// 处理邮件正文
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// 处理内联内容
			content, err := ioutil.ReadAll(p.Body)
			if err != nil {
				continue
			}
			ct, _, _ := h.ContentType()
			switch ct {
			case "text/plain":
				email.Text += string(content) + "\n"
			case "text/html":
				email.HTML += string(content)
			}
		}
	}

	if email.HTML != "" {
		email.Text += extractTextFromHTML(email.HTML)
	}

	return email, nil
}

// 解码邮件头
func decodeHeader(header string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		return header
	}
	return decoded
}

// extractTextFromHTML 从 HTML 内容中提取纯文本
func extractTextFromHTML(htmlContent string) string {
	// 解析 HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		fmt.Println("解析 HTML 失败:", err)
		return ""
	}

	var buf bytes.Buffer
	var extractText func(*html.Node)

	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(strings.TrimSpace(n.Data) + " ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)

	return strings.TrimSpace(buf.String())
}
