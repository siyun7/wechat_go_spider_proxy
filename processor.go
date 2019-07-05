package wegospider

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/palantir/stacktrace"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Processor interface {
	ProcessList(resp *http.Response, ctx *goproxy.ProxyCtx) ([]byte, error)
	ProcessDetail(resp *http.Response, ctx *goproxy.ProxyCtx) ([]byte, error)
	ProcessMetrics(resp *http.Response, ctx *goproxy.ProxyCtx) ([]byte, error)
	ProcessPages() error
	Output()
}

type BaseProcessor struct {
	req				*http.Request
	lastId			string
	data			[]byte
	urlResults		[]*UrlResult
	detailResult	*DetailResult
	historyUrl		string
	biz				string

	currentIndex int
	checked      bool
	Type         string
}

type UrlResult struct {
	Mid				string
	Url				string
	_URL			*url.URL
}

type MagStat struct {
	ReadNum		int `json:"read_num"`
	LikeNum		int `json:"like_num"`
	RealReadNum	int `json:"real_read_num"`
}

type DetailResult struct {
	Id			string
	Url			string
	Data		[]byte
	Appmsgstat	*MagStat `json:"appmsgstat"`
}

var (
	replacer = strings.NewReplacer(
		"\t", "", " ", "",
		"&quot;", `"`, "&nbsp;", "",
		`\\`, "", "&amp;amp;", "&",
		"&amp;", "&", `\`, "",
	)
	urlRegex 	= regexp.MustCompile(`http://mp.weixin.qq.com/s?[^#"',]*`)
	idRegex		= regexp.MustCompile(`"id":(\d+)`)
	MsgNotFound = errors.New("MagLists not found")

	TypeList	= "list"
	TypeDetail	= "detail"
	TypeMetric	= "metric"
)

func NewBaseProcessor() *BaseProcessor {
	return &BaseProcessor{}
}

func (p *BaseProcessor) init(req *http.Request, data []byte) (err error) {
	p.req = req
	p.data = data
	p.currentIndex = -1
	p.biz = req.URL.Query().Get("__biz")
	p.logf("Running a new wechat processor, please wait...")
	return nil
}

func (p *BaseProcessor) UrlResults() []*UrlResult {
	return p.urlResults
}

func (p *BaseProcessor) DetailResults() *DetailResult {
	return p.detailResult
}

func (p *BaseProcessor) GetRequest() *http.Request {
	return p.req
}

func (p *BaseProcessor) logf(format string, msg ...interface{}) {
	if rootConfig.Debug {
		Logger.Printf(format, msg...)
	}
}

func (p *BaseProcessor) ProcessList(resp *http.Response, ctx *goproxy.ProxyCtx) (data []byte, err error) {
	p.Type = TypeList
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return
	}
	if err = resp.Body.Close(); err != nil {
		return
	}

	data = buf.Bytes()
	if err = p.init(ctx.Req, data); err != nil {
		return
	}
	if err = p.processMain(); err != nil{
		return
	}
	return
}
func (p *BaseProcessor) ProcessDetail(resp *http.Response, ctx *goproxy.ProxyCtx) (data []byte, err error) {
	return
}

func (p *BaseProcessor) ProcessMetrics(resp *http.Response, ctx *goproxy.ProxyCtx) (data []byte, err error) {
	return
}

func (p *BaseProcessor) Output() {
	var urls []string
	fmt.Println("result => [")
	for _, r := range p.urlResults{
		urls = append(urls, r.Url)
	}
	fmt.Println(strings.Join(urls, ","))
	fmt.Println("]")
}

//ProcessPages AutoScrool page
func (p *BaseProcessor) ProcessPages() (err error) {
	if err = p.sendCheckUrl(); err != nil {
		return
	}
	var pageUrl = p.genPageUrl()
	p.logf("process next pages...")
	req, err := http.NewRequest("GET", pageUrl, nil)
	if err != nil {
		return stacktrace.Propagate(err, "Failed new page request")
	}
	for k := range p.req.Header {
		req.Header.Set(k, p.req.Header.Get(k))
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return stacktrace.Propagate(err, "Failed get page response")
	}
	bs, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	str := replacer.Replace(string(bs))
	resutl := urlRegex.FindAllString(str, -1)
	if len(resutl) < 1 {
		return stacktrace.Propagate(err, "Failed get page url")
	}
	idMatcher := idRegex.FindAllStringSubmatch(str, -1)
	if len(idMatcher) < 1 {
		return stacktrace.Propagate(err, "Failed get page id")
	}
	lastId := idMatcher[len(idMatcher)-1][1]
	for _, u :=  range resutl {
		p.urlResults = append(p.urlResults, &UrlResult{Url: u})
	}
	if lastId != ""{
		if p.lastId == lastId {
			i, _ := strconv.Atoi(lastId)
			p.lastId = fmt.Sprintf("%d", i-10)
		} else {
			p.lastId = lastId
		}
		p.Sleep()
		return p.ProcessPages()
	}
	return nil
}

func (p *BaseProcessor) Sleep() {
	ti := rand.Intn(rootConfig.SleepSecond) + 1
	time.Sleep(time.Duration(ti) * time.Second)
}

func (p *BaseProcessor) processMain() error {
	p.urlResults = make([]*UrlResult, 0, 100)
	buffer := bytes.NewBuffer(p.data)
	var msgs string
	str, err := buffer.ReadString('\n')
	for err == nil {
		if strings.Contains(str, "msgList = ") {
			msgs = str
			break
		}
		str, err = buffer.ReadString('\n')
	}
	if msgs == "" {
		return stacktrace.Propagate(MsgNotFound, "Failed parse main")
	}
	msgs = replacer.Replace(msgs)
	urls := urlRegex.FindAllString(msgs, -1)
	if len(urls) < 1 {
		return stacktrace.Propagate(MsgNotFound, "Failed parse main")
	}
	p.urlResults = make([]*UrlResult, len(urls))
	for i, u := range urls{
		p.urlResults[i] = &UrlResult{Url:u}
	}
	idMatcher := idRegex.FindAllStringSubmatch(msgs, -1)
	if len(idMatcher) < 1 {
		return stacktrace.Propagate(MsgNotFound, "Failed find id in main")
	}
	p.lastId = idMatcher[len(idMatcher)-1][1]
	return nil
}

func genId(urlStr string) string {
	uri, _ :=url.ParseRequestURI(urlStr)
	return hashKey(uri.Query().Get("__biz") + "_" + uri.Query().Get("mid") + "_" + uri.Query().Get("idx"))
}

func hashKey(key string) string {
	h := md5.New()
	io.WriteString(h, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *BaseProcessor) genPageUrl() string {
	p.logf("loading pages, urls size now is %d", len(p.UrlResults()))
	otherQuery := strings.Replace(p.req.URL.RawQuery, "action=home", "", -1)
	return fmt.Sprintf("https://mp.weixin.qq.com/mp/profile_ext?%s&frommsgid=%s&f=json&count=10&is_ok=1&action=getmsg&f=json&wxtoken=&x5=1&uin=777&key=777", otherQuery, p.lastId)
}

func (p *BaseProcessor) sendCheckUrl() (err error) {
	if p.checked {
		return nil
	}
	p.checked = true
	values := url.Values{}
	query := p.req.URL.Query()
	values.Add("__biz", query.Get("__biz"))
	values.Add("scene", query.Get("scene"))
	values.Add("url_list", "")
	urlStr := fmt.Sprintf("http://mp.weixin.qq.com/mp/profile_ext?action=urlcheck&uin=%s&key=%s&pass_ticket=%s", query.Get("uin"), query.Get("key"), query.Get("pass_ticket"))

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(values.Encode()))
	if err != nil {
		return stacktrace.Propagate(err, "Failed check request")
	}

	for k := range p.req.Header {
		req.Header.Set(k, p.req.Header.Get(k))
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}