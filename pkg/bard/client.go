package bard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Vaayne/aienvoy/pkg/session"
	utls "github.com/refraction-networking/utls"
)

const (
	bardUrl        = "https://bard.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	cookieTokenKey = "__Secure-1PSID"
)

var headers map[string]string = map[string]string{
	"Host":          "bard.google.com",
	"X-Same-Domain": "1",
	"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Content-Type":  "application/x-www-form-urlencoded;charset=utf-8",
	"Origin":        "https://bard.google.com",
	"Referer":       "https://bard.google.com/",
}

type Client struct {
	token   string
	snlm0e  string
	cfb2h   string
	cookies map[string]string
	timeout time.Duration
	session *session.Session
}

type ClientOption func(*Client)

func NewBardClient(token string, opts ...ClientOption) (*Client, error) {
	if token == "" || !strings.HasSuffix(token, ".") {
		return nil, fmt.Errorf("__Secure-1PSID value must end with a single dot. Enter correct __Secure-1PSID value: %s", token)
	}

	jar, _ := cookiejar.New(nil)
	b := &Client{
		token:   token,
		session: session.New(session.WithCookieJar(jar), session.WithClientHelloID(utls.HelloChrome_100_PSK)),
	}

	for _, opt := range opts {
		opt(b)
	}

	// set cookies
	setCookie := func(key, val string) *http.Cookie {
		return &http.Cookie{
			Name:   key,
			Value:  val,
			Domain: "google.com",
		}
	}

	cookies := []*http.Cookie{
		setCookie(cookieTokenKey, b.token),
	}
	for key, val := range b.cookies {
		cookies = append(cookies, setCookie(key, val))
	}
	jar.SetCookies(&url.URL{
		Scheme: "https",
		Host:   "bard.google.com",
	}, cookies)

	err := b.initMeta()
	return b, err
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(bc *Client) {
		bc.timeout = timeout
	}
}

func WithCookies(cookies map[string]string) ClientOption {
	return func(bc *Client) {
		bc.cookies = cookies
	}
}

func (b *Client) initMeta() error {
	req, err := http.NewRequest(http.MethodGet, "https://bard.google.com/", nil)
	if err != nil {
		return fmt.Errorf("init request for SNlM0e error: %w", err)
	}
	b.setHeaders(req)

	resp, err := b.session.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Response code not 200. Response Status is %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	extractFromHTML := func(name string) (string, error) {
		re := regexp.MustCompile(fmt.Sprintf(`%s":"(.*?)"`, name))
		matches := re.FindStringSubmatch(string(body))
		if len(matches) != 2 {
			return "", fmt.Errorf(name + " value not found in response. Check __Secure-1PSID value.")
		}
		return matches[1], nil
	}

	if snlm0e, err := extractFromHTML("SNlM0e"); err != nil {
		return err
	} else {
		b.snlm0e = snlm0e
	}

	if cfb2h, err := extractFromHTML("cfb2h"); err != nil {
		return err
	} else {
		b.cfb2h = cfb2h
	}
	return nil
}

func (b *Client) FirstAsk(prompt string) (*Answer, error) {
	return b.Ask(prompt, "", "", "", 0)
}

func (b *Client) Ask(prompt, conversationID, responseID, choiceID string, reqID int) (*Answer, error) {
	// b.mu.Lock()
	// defer b.mu.Unlock()
	req, err := b.buildRequest(prompt, conversationID, responseID, choiceID, reqID)
	if err != nil {
		return nil, fmt.Errorf("build bard request error: %w", err)
	}
	resp, err := b.session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to bard error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read bard response body error: %w", err)
	}

	data1 := bytes.Split(body, []byte("\n"))
	return parse(string(data1[3]))
}

func (b *Client) setHeaders(req *http.Request) {
	for key, val := range headers {
		req.Header.Add(key, val)
	}
}

func (b *Client) buildRequest(prompt, conversationID, responseID, choiceID string, reqID int) (*http.Request, error) {
	// build req url
	if reqID == 0 {
		reqID = 100000 + rand.Intn(10000)
	}
	params := url.Values{
		"bl":     {b.cfb2h},
		"_reqid": {strconv.Itoa(reqID)},
		"rt":     {"c"},
	}
	reqURL := bardUrl + "?" + params.Encode()

	// build req body
	inputTextStruct := [][]any{
		{prompt},
		nil,
		{conversationID, responseID, choiceID},
	}

	inputText, err := json.Marshal(inputTextStruct)
	if err != nil {
		return nil, fmt.Errorf("encode input text error: %w", err)
	}

	data := []any{nil, string(inputText)}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode input data error: %w", err)
	}

	reqData := url.Values{
		"f.req": {string(jsonData)},
		"at":    {b.snlm0e},
	}
	// new http request
	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(reqData.Encode()))
	if err != nil {
		return nil, err
	}
	// set header
	b.setHeaders(req)

	return req, nil
}
