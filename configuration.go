package wegospider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type MongoConfiguration struct {
	Host string
	Port string
}

type SpiderConfiguration struct {
	Debug bool
	AutoScrool bool
	Compress bool
	SleepSecond int
	ProxyPort string
}

type Configuration struct {
	Mongo MongoConfiguration
	Spider SpiderConfiguration
}


func (c *Configuration) LoadFromFile() error {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fileName := filepath.Join(dir, "conf/main.json")
	file, _ := os.Open(fileName)
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&c)
	fmt.Printf("%+v\n", *c)
	if err != nil {
		return err
	}
	return nil
}
