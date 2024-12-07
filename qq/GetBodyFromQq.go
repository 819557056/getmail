package qq

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/joho/godotenv"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// 转换字符编码
func convertToUTF8(content []byte, charset string) (string, error) {
	charset = strings.ToLower(charset)
	var transformer transform.Transformer

	switch charset {
	case "gb2312", "gb18030":
		transformer = simplifiedchinese.GB18030.NewDecoder()
	case "big5":
		transformer = traditionalchinese.Big5.NewDecoder()
	case "euc-jp":
		transformer = japanese.EUCJP.NewDecoder()
	case "shift-jis":
		transformer = japanese.ShiftJIS.NewDecoder()
	case "euc-kr":
		transformer = korean.EUCKR.NewDecoder()
	case "utf-8", "":
		return string(content), nil
	default:
		return string(content), fmt.Errorf("unsupported charset: %s", charset)
	}

	reader := transform.NewReader(bytes.NewReader(content), transformer)
	decoded, err := ioutil.ReadAll(reader)
	if err != nil {
		return string(content), err
	}
	return string(decoded), nil
}

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

	// 获取最新的邮件
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(mbox.Messages)

	// 获取邮件内容
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			log.Fatal("服务器未返回邮件内容")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
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
			Date:    date.String()}

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			// 读取原始内容
			b, err := ioutil.ReadAll(p.Body)
			if err != nil {
				continue
			}

			var mediaType string
			var charset string

			// 根据不同的header类型处理
			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				if ct, params, err := h.ContentType(); err == nil {
					mediaType = ct
					charset = params["charset"]
				}
			case *mail.AttachmentHeader:
				if ct, params, err := h.ContentType(); err == nil {
					mediaType = ct
					charset = params["charset"]
				}
			}

			// 获取编码方式
			encoding := p.Header.Get("Content-Transfer-Encoding")

			// 打印调试信息
			fmt.Printf("Debug - MediaType: %s, Charset: %s, Encoding: %s, Raw Content Length: %d\n",
				mediaType, charset, encoding, len(b))

			// 处理内容编码
			var content []byte
			switch strings.ToLower(encoding) {
			case "base64":
				decoded, err := base64.StdEncoding.DecodeString(string(b))
				if err == nil {
					content = decoded
				} else {
					content = b
				}
			case "quoted-printable":
				// 如果需要处理 quoted-printable编码
				content = b
			default:
				content = b
			}

			// 处理字符集
			var decodedContent string
			if charset != "" {
				switch strings.ToLower(charset) {
				case "gb2312", "gb18030":
					reader := transform.NewReader(bytes.NewReader(content), simplifiedchinese.GB18030.NewDecoder())
					if decoded, err := ioutil.ReadAll(reader); err == nil {
						decodedContent = string(decoded)
					}
				case "gbk":
					reader := transform.NewReader(bytes.NewReader(content), simplifiedchinese.GBK.NewDecoder())
					if decoded, err := ioutil.ReadAll(reader); err == nil {
						decodedContent = string(decoded)
					}
				default:
					decodedContent = string(content)
				}
			} else {
				decodedContent = string(content)
			}

			// 存储内容
			switch mediaType {
			case "text/plain":
				email.Text = decodedContent
			case "text/html":
				email.HTML = decodedContent
			}
		}

		// 打印邮件信息
		fmt.Printf("\n=== 邮件详情 ===\n")
		fmt.Printf("主题: %v\n", email.Subject)
		fmt.Printf("发件人: %v\n", email.From)
		fmt.Printf("收件人: %v\n", email.To)
		fmt.Printf("日期: %v\n", email.Date)
		if email.Text != "" {
			fmt.Printf("\n=== 文本内容 ===\n%v\n", email.Text)
		}
		if email.HTML != "" {
			fmt.Printf("\n=== HTML内容 ===\n%v\n", email.HTML)
			fromHTML := extractTextFromHTML(email.HTML)
			fmt.Printf("\n=== 文本内容 ===\n%v\n", fromHTML)
		}
	}
	if err := <-done; err != nil {
		log.Fatal(err)
	}
}

// 检查是否是有效的 UTF-8
func isUTF8(buf []byte) bool {
	return strings.Contains(http.DetectContentType(buf), "charset=utf-8")
}

// 尝试不同的编码
func tryDifferentEncodings(content []byte) string {
	//尝试 UTF-8
	if isUTF8(content) {
		return string(content)
	}

	// 尝试 GB18030
	reader := transform.NewReader(bytes.NewReader(content), simplifiedchinese.GB18030.NewDecoder())
	if decoded, err := ioutil.ReadAll(reader); err == nil {
		return string(decoded)
	}

	// 尝试 GBK
	reader = transform.NewReader(bytes.NewReader(content), simplifiedchinese.GBK.NewDecoder())
	if decoded, err := ioutil.ReadAll(reader); err == nil {
		return string(decoded)
	}

	// 如果都失败了，返回原始内容
	return string(content)
}

func getUnreadEmailsForRecipient(c *client.Client, recipientEmail string) ([]*EmailContent, error) {
	// 选择收件箱
	_, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("选择收件箱失败: %v", err)
	}

	// 搜索未读邮件
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	//criteria.To = []string{recipientEmail}

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
