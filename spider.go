package wegospider

import (
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"strings"
)

type spider struct {
	proxy *goproxy.ProxyHttpServer
}

type Config struct {
	Debug bool
	AutoScrool bool
	Compress bool
	SleepSecond int
}

var _spider  = newSpider()

var (
	header http.Header

	rootConfig = &Config{
		Debug:			false,
		AutoScrool:		false,
		Compress:		true,
		SleepSecond:	3,
	}
)

func newSpider() *spider {
	s := &spider{}
	s.proxy = goproxy.NewProxyHttpServer()

	s.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	return s
}

func InitConfig(conf *Config)  {
	rootConfig = conf
	if rootConfig.SleepSecond < 1 {
		rootConfig.SleepSecond = 3
	}
}

func GetConfig() *Config {
	return rootConfig
}

func Regist(proc Processor)  {
	_spider.Regist(proc)
}

func (s *spider) Regist(proc Processor) {
	s.proxy.OnResponse().DoFunc(ProxyHandle(proc))
}

func Run(port string)  {
	_spider.Run(port)
}

func (s *spider) Run(port string) {
	if rootConfig.Compress {
		s.OnReq(func(ctx *goproxy.ProxyCtx) (req *http.Request, resp *http.Response) {
			host := ctx.Req.URL.Host
			req = ctx.Req
			if !strings.Contains(host, "mp.weixin.qq.com") {
				resp = goproxy.NewResponse(req, "text/html", http.StatusNotFound, "")
			}
			return
		})
	}
	log.Println("Server listening port:" + port)
	log.Fatal(http.ListenAndServe(":" + port, s.proxy))
}

func (s *spider) OnReq(f func(ctx *goproxy.ProxyCtx) (*http.Request, *http.Response)) {
	handler := func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return f(ctx)
	}
	s.proxy.OnRequest().DoFunc(handler)
}