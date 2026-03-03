package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url" // 关键：用于解析代理协议
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	CorpID         string `json:"corpid"`
	AgentID        string `json:"agentid"`
	CorpSecret     string `json:"corpsecret"`
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
	ProxyURL       string `json:"proxy_url"`
	NasURL         string `json:"nas_url"`
	PhotoURL       string `json:"photo_url"`
	Configured     bool   `json:"configured"`
}

var (
	configPath           = "data/config.json"
	accessToken          string
	accessTokenExpiresAt int64
	sessionToken         string
	Version              = "v5.0.0-Final"
)

func init() {
	b := make([]byte, 16)
	rand.Read(b)
	sessionToken = hex.EncodeToString(b)
}

func main() {
	_ = os.MkdirAll("data", 0755)
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 1. 模板与静态资源配置
	r.LoadHTMLGlob("templates/*")
	// 修复：显式映射根目录的 logo.png，确保网页能正常显示
	r.StaticFile("/logo.png", "./logo.png")
	// 如果你还有其他静态文件，可以保留这个映射
	r.Static("/static", "./static")

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
	}

	// 2. 身份验证路由
	r.GET("/login", func(c *gin.Context) {
		if checkCookie(c) {
			c.Redirect(http.StatusFound, "/")
			return
		}
		c.HTML(http.StatusOK, "login.html", gin.H{"version": Version})
	})

	r.POST("/login", func(c *gin.Context) {
		if c.PostForm("password") == adminPass {
			c.SetCookie("auth_session", sessionToken, 86400, "/", "", false, true)
			c.Redirect(http.StatusFound, "/")
		} else {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "密码错误", "version": Version})
		}
	})

	r.GET("/logout", func(c *gin.Context) {
		c.SetCookie("auth_session", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
	})

	// 3. Webhook 接口
	r.GET("/webhook", handleVerify)
	r.POST("/webhook", handleMessage)

	// 4. 管理后台
	auth := r.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"config":  loadConfig(),
				"success": c.Query("success"),
				"version": Version,
			})
		})
		auth.POST("/save", handleSave)
	}

	log.Printf("NAS Webhook %s Started on :5080", Version)
	r.Run(":5080")
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkCookie(c) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func checkCookie(c *gin.Context) bool {
	cookie, _ := c.Cookie("auth_session")
	return cookie == sessionToken
}

func loadConfig() Config {
	var conf Config
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &conf)
	}
	return conf
}

func handleSave(c *gin.Context) {
	conf := Config{
		CorpID:         c.PostForm("corpid"),
		AgentID:        c.PostForm("agentid"),
		CorpSecret:     c.PostForm("corpsecret"),
		Token:          c.PostForm("token"),
		EncodingAESKey: c.PostForm("encoding_aes_key"),
		ProxyURL:       strings.TrimSpace(c.PostForm("proxy_url")),
		NasURL:         strings.TrimRight(c.PostForm("nas_url"), "/"),
		PhotoURL:       c.PostForm("photo_url"),
		Configured:     true,
	}
	data, _ := json.MarshalIndent(conf, "", "  ")
	os.WriteFile(configPath, data, 0644)
	c.Redirect(http.StatusSeeOther, "/?success=true")
}

func handleVerify(c *gin.Context) {
	conf := loadConfig()
	msgSig := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	params := []string{conf.Token, timestamp, nonce, echostr}
	sort.Strings(params)
	h := sha1.New()
	h.Write([]byte(strings.Join(params, "")))
	if fmt.Sprintf("%x", h.Sum(nil)) != msgSig {
		c.AbortWithStatus(403)
		return
	}

	aesKey, _ := base64.StdEncoding.DecodeString(conf.EncodingAESKey + "=")
	cipherText, _ := base64.StdEncoding.DecodeString(echostr)
	block, _ := aes.NewCipher(aesKey)
	mode := cipher.NewCBCDecrypter(block, aesKey[:16])
	mode.CryptBlocks(cipherText, cipherText)
	msgLen := binary.BigEndian.Uint32(cipherText[16:20])
	c.String(200, string(cipherText[20:20+msgLen]))
}

func handleMessage(c *gin.Context) {
	// 超级解析：兼容 URL 传参和 JSON Body
	data := make(map[string]interface{})
	if txt := c.Query("text"); txt != "" { data["text"] = txt }
	
	var jsonData map[string]interface{}
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &jsonData)
		for k, v := range jsonData { data[k] = v }
	}

	if len(data) > 0 {
		conf := loadConfig()
		if conf.Configured {
			log.Printf("[Webhook] 接收到推送数据: %v", data)
			go pushToWeChat(conf, data)
		}
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func pushToWeChat(conf Config, data map[string]interface{}) {
	// 构建支持可选代理的 HTTP Client
	transport := &http.Transport{}
	if conf.ProxyURL != "" {
		proxy, err := url.Parse(conf.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxy)
			log.Printf("[Proxy] 代理启用: %s", conf.ProxyURL)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	token := getWeChatToken(conf, client)
	if token == "" { return }

	content := "NAS 系统通知"
	if t, ok := data["text"].(string); ok { content = t }
	if m, ok := data["message"].(string); ok { content = m }

	picURL := conf.PhotoURL
	if picURL == "" {
		picURL = fmt.Sprintf("https://picsum.photos/600/300?random=%d", time.Now().Unix())
	}

	agentID, _ := strconv.Atoi(conf.AgentID)
	payload := map[string]interface{}{
		"touser":  "@all",
		"msgtype": "news",
		"agentid": agentID,
		"news": map[string]interface{}{
			"articles": []map[string]interface{}{{
				"title":       "NAS 通知中心",
				"description": fmt.Sprintf("时间: %s\n内容: %s", time.Now().Format("15:04:05"), content),
				"url":         conf.NasURL,
				"picurl":      picURL,
			}},
		},
	}

	body, _ := json.Marshal(payload)
	// 使用带代理逻辑的 client 发送
	resp, err := client.Post("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+token, "application/json", bytes.NewBuffer(body))
	if err == nil {
		res, _ := io.ReadAll(resp.Body)
		log.Printf("[WeChat] 发送状态: %s", string(res))
		resp.Body.Close()
	} else {
		log.Printf("[WeChat] 网络故障: %v", err)
	}
}

func getWeChatToken(conf Config, client *http.Client) string {
	if accessToken != "" && accessTokenExpiresAt > time.Now().Unix() {
		return accessToken
	}
	resp, err := client.Get(fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", conf.CorpID, conf.CorpSecret))
	if err != nil {
		log.Printf("[Token] 错误: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var res struct {
		Token string `json:"access_token"`
		Exp   int64  `json:"expires_in"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	accessToken = res.Token
	accessTokenExpiresAt = time.Now().Unix() + res.Exp - 60
	return accessToken
}