package main

import (
	"encoding/base64"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/url"
	"strings"
	spider "wegospider"
)



func main()  {
	var port = app.Configuration.Spider.ProxyPort
	print(port)
	spider.InitConfig(&spider.Config{
		Debug:app.Configuration.Spider.Debug,
		AutoScrool:app.Configuration.Spider.AutoScrool,
		Compress:app.Configuration.Spider.Compress,
	})
	spider.Regist(&CustomProcessor{})
	spider.Run(port)
}

type CustomProcessor struct {
	spider.BaseProcessor
}

type App struct {
	Configuration *spider.Configuration
}

type M map[string]interface{}

func (c *CustomProcessor) Output() {
	collection := "wx_article"
	type Url struct {
		NUM int "bson: `num`"
	}
	last := Url{}
	db.C(collection).Find(bson.M{}).Sort("-num").One(&last)
	for key, result := range c.UrlResults() {
		uri, _ := url.ParseRequestURI(result.Url)
		id := strings.Join([]string{uri.Query().Get("__biz"), uri.Query().Get("mid"), uri.Query().Get("idx")}, "_")
		encodeId := base64.StdEncoding.EncodeToString([]byte(id))
		num := last.NUM + key +1
		//fmt.Println(encodeId, "   Url:", result.Url)
		exist, err := db.C("wx_article").Find(bson.M{"_id": encodeId}).Count()
		if err != nil {
			panic(err)
		}
		if exist != 0 {
			continue
		}
		db.C("wx_article").UpsertId(encodeId, M{"url": result.Url, "num": num, "usage": 0})
	}
	fmt.Println("公众号抓取完成！")
}

var (
	db *mgo.Database
	app *App
)

func init()  {
	app = &App{}
	app.Configuration = &spider.Configuration{}
	app.Configuration.LoadFromFile()
	tmp := []string{app.Configuration.Mongo.Host, app.Configuration.Mongo.Port}
	mongoServer := strings.Join(tmp, ":")
	session, err := mgo.Dial(mongoServer)
	if err != nil {
		panic(err)
	}
	db = session.DB("wechat")
}


