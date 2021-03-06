//Copyright 2017-2018 Baidu Inc.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http: //www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package models

import (
	"fmt"
	"crypto/md5"
	"rasp-cloud/mongo"
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
	"sync"
	"time"
	"regexp"
	"encoding/json"
	"errors"
	"strconv"
	"math/rand"
	"crypto/sha1"
	"rasp-cloud/tools"
	"gopkg.in/mgo.v2"
	"bufio"
	"bytes"
	"github.com/robertkrimen/otto"
)

type Plugin struct {
	Id                     string                 `json:"id" bson:"_id,omitempty"`
	AppId                  string                 `json:"app_id" bson:"app_id"`
	Name                   string                 `json:"name" bson:"name"`
	UploadTime             int64                  `json:"upload_time" bson:"upload_time"`
	Version                string                 `json:"version" bson:"version"`
	Md5                    string                 `json:"md5" bson:"md5"`
	Content                string                 `json:"plugin,omitempty" bson:"content"`
	DefaultAlgorithmConfig map[string]interface{} `bson:"default_algorithm_config"`
	AlgorithmConfig        map[string]interface{} `json:"algorithm_config" bson:"algorithm_config"`
}

const (
	pluginCollectionName = "plugin"
)

var (
	mutex      sync.Mutex
	MaxPlugins int
)

func init() {
	if value, err := beego.AppConfig.Int("MaxPlugins"); err != nil || value <= 0 {
		tools.Panic(tools.ErrCodeMongoInitFailed, "the 'AlarmBufferSize' config must be greater than 0", nil)
	} else if value < 10 {
		beego.Warning("the value of 'MaxPlugins' config is less than 10, it will be set to 10")
		MaxPlugins = 10
	} else {
		MaxPlugins = value
	}
	count, err := mongo.Count(pluginCollectionName)
	if err != nil {
		tools.Panic(tools.ErrCodeMongoInitFailed, "failed to get plugin collection count", err)
	}
	if count <= 0 {
		index := &mgo.Index{
			Key:        []string{"app_id"},
			Unique:     false,
			Background: true,
			Name:       "app_id",
		}
		err := mongo.CreateIndex(pluginCollectionName, index)
		if err != nil {
			tools.Panic(tools.ErrCodeMongoInitFailed,
				"failed to create app_id index for plugin collection", err)
		}
		index = &mgo.Index{
			Key:        []string{"upload_time"},
			Unique:     false,
			Background: true,
			Name:       "upload_time",
		}
		err = mongo.CreateIndex(pluginCollectionName, index)
		if err != nil {
			tools.Panic(tools.ErrCodeMongoInitFailed,
				"failed to create upload_time index for plugin collection", err)
		}
	}
}

func AddPlugin(pluginContent []byte, appId string) (plugin *Plugin, err error) {

	pluginReader := bufio.NewReader(bytes.NewReader(pluginContent))
	firstLine, err := pluginReader.ReadString('\n')
	if err != nil {
		return nil, errors.New("failed to read the plugin file: " + err.Error())
	}
	secondLine, err := pluginReader.ReadString('\n')
	if err != nil {
		return nil, errors.New("failed to read the plugin file: " + err.Error())
	}
	var newVersion string
	var newPluginName string
	if newVersion = regexp.MustCompile(`'.+'|".+"`).FindString(firstLine); newVersion == "" {
		return nil, errors.New("failed to find the plugin version")
	}
	newVersion = newVersion[1 : len(newVersion)-1]
	if newPluginName = regexp.MustCompile(`'.+'|".+"`).FindString(secondLine); newPluginName == "" {
		return nil, errors.New("failed to find the plugin name")
	}
	newPluginName = newPluginName[1 : len(newPluginName)-1]
	algorithmStartMsg := "// BEGIN ALGORITHM CONFIG //"
	algorithmEndMsg := "// END ALGORITHM CONFIG //"
	algorithmStart := bytes.Index(pluginContent, []byte(algorithmStartMsg))
	if algorithmStart < 0 {
		return nil, errors.New("failed to find the start of algorithmConfig variable: " + algorithmStartMsg)
	}
	algorithmStart = algorithmStart + len([]byte(algorithmStartMsg))
	algorithmEnd := bytes.Index(pluginContent, []byte(algorithmEndMsg))
	if algorithmEnd < 0 {
		return nil, errors.New("failed to find the end of algorithmConfig variable: " + algorithmEndMsg)
	}
	jsVm := otto.New()
	_, err = jsVm.Run(string(pluginContent[algorithmStart:algorithmEnd]) +
		"\n algorithmContent=JSON.stringify(algorithmConfig)")
	if err != nil {
		return nil, errors.New("failed to get algorithm config from plugin: " + err.Error())
	}
	algorithmContent, err := jsVm.Get("algorithmContent")
	if err != nil {
		return nil, errors.New("failed to get algorithm config from plugin: " + err.Error())
	}
	var algorithmData map[string]interface{}
	err = json.Unmarshal([]byte(algorithmContent.String()), &algorithmData)
	if err != nil {
		return nil, errors.New("failed to unmarshal algorithm json data: " + err.Error())
	}
	return addPluginToDb(newVersion, newPluginName, pluginContent, appId, algorithmData)

}

func addPluginToDb(version string, name string, content []byte, appId string,
	defaultAlgorithmConfig map[string]interface{}) (plugin *Plugin, err error) {
	newMd5 := fmt.Sprintf("%x", md5.Sum(content))
	plugin = &Plugin{
		Id:                     generatePluginId(appId),
		Version:                version,
		Name:                   name,
		Md5:                    newMd5,
		Content:                string(content),
		UploadTime:             time.Now().UnixNano() / 1000000,
		AppId:                  appId,
		DefaultAlgorithmConfig: defaultAlgorithmConfig,
		AlgorithmConfig:        defaultAlgorithmConfig,
	}
	mutex.Lock()
	defer mutex.Unlock()

	var count int
	if MaxPlugins > 0 {
		_, oldPlugins, err := GetPluginsByApp(appId, MaxPlugins-1, 0)
		if err != nil {
			return nil, err
		}
		count = len(oldPlugins)
		if count > 0 {
			for _, oldPlugin := range oldPlugins {
				err = mongo.RemoveId(pluginCollectionName, oldPlugin.Id)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	err = mongo.Insert(pluginCollectionName, plugin)
	return
}

func generatePluginId(appId string) string {
	random := string(bson.NewObjectId()) + appId +
		strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(10000))
	return fmt.Sprintf("%x", sha1.Sum([]byte(random)))
}

func GetSelectedPlugin(appId string, hasContent bool) (plugin *Plugin, err error) {
	var app *App
	if err = mongo.FindId(appCollectionName, appId, &app); err != nil {
		return
	}
	return GetPluginById(app.SelectedPluginId, hasContent)
}

func SetSelectedPlugin(appId string, pluginId string) error {
	_, err := GetPluginById(pluginId, false)
	if err != nil {
		return err
	}
	return mongo.UpdateId(appCollectionName, appId, bson.M{"selected_plugin_id": pluginId})
}

func RestoreDefaultConfiguration(pluginId string) (appId string, err error) {
	plugin, err := GetPluginById(pluginId, true)
	if err != nil {
		return "", err
	}
	return handleAlgorithmConfig(plugin, plugin.DefaultAlgorithmConfig)
}

func UpdateAlgorithmConfig(pluginId string, config map[string]interface{}) (appId string, err error) {
	plugin, err := GetPluginById(pluginId, true)
	if err != nil {
		return "", err
	}
	if err := validAlgorithmConfig(plugin, config); err != nil {
		return "", err
	}
	return handleAlgorithmConfig(plugin, config)
}

func validAlgorithmConfig(plugin *Plugin, config map[string]interface{}) error {
	errMsg := "failed to match the new config format to default algorithm config format"
	for key, defaultValue := range plugin.DefaultAlgorithmConfig {
		if c, ok := config[key]; !ok || (c == nil && defaultValue != nil) {
			return errors.New(errMsg + ", " + "can not find the key '" + key + "' in new config")
		}
		if defaultValue != nil {
			if defaultItem, ok := defaultValue.(map[string]interface{}); ok {
				if item, ok := config[key].(map[string]interface{}); ok {
					for subKey := range defaultItem {
						if _, ok := item[subKey]; !ok {
							return errors.New(errMsg + ", " + "can not find the key '" +
								key + "." + subKey + "' in new config")
						}
					}
				} else {
					return errors.New(errMsg + ", " + "the key '" + key + "' must be an object")
				}
			}
		}
	}
	return nil
}

func handleAlgorithmConfig(plugin *Plugin, config map[string]interface{}) (appId string, err error) {
	content, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return "", err
	}
	regex := `//\s*BEGIN\s*ALGORITHM\s*CONFIG\s*//[\W\w]*?//\s*END\s*ALGORITHM\s*CONFIG\s*//`
	newContent := "// BEGIN ALGORITHM CONFIG //\n\n" +
		"var algorithmConfig = " +
		string(content) + "\n\n// END ALGORITHM CONFIG //"
	if variable := regexp.MustCompile(regex).
		FindString(plugin.Content); len(variable) <= 0 {
		return "", errors.New("failed to find algorithmConfig variable")
	}
	algorithmContent := regexp.MustCompile(regex).ReplaceAllString(plugin.Content, newContent)
	newMd5 := fmt.Sprintf("%x", md5.Sum([]byte(algorithmContent)))
	fmt.Println(algorithmContent)
	return plugin.AppId, mongo.UpdateId(pluginCollectionName, plugin.Id, bson.M{"content": algorithmContent,
		"algorithm_config": config, "md5": newMd5})
}

func GetPluginById(id string, hasContent bool) (plugin *Plugin, err error) {
	newSession := mongo.NewSession()
	defer newSession.Close()
	query := newSession.DB(mongo.DbName).C(pluginCollectionName).FindId(id)
	if hasContent {
		err = query.One(&plugin)
	} else {
		err = query.Select(bson.M{"content": 0}).One(&plugin)
	}
	return
}

func GetPluginsByApp(appId string, skip int, limit int) (total int, plugins []Plugin, err error) {
	newSession := mongo.NewSession()
	defer newSession.Close()
	total, err = newSession.DB(mongo.DbName).C(pluginCollectionName).Find(bson.M{"app_id": appId}).Count()
	if err != nil {
		return
	}
	err = newSession.DB(mongo.DbName).C(pluginCollectionName).Find(bson.M{"app_id": appId}).Select(bson.M{"content": 0}).
		Sort("-upload_time").Skip(skip).Limit(limit).All(&plugins)
	if plugins == nil {
		plugins = make([]Plugin, 0)
	}
	return
}

func DeletePlugin(pluginId string) error {
	mutex.Lock()
	defer mutex.Unlock()
	return mongo.RemoveId(pluginCollectionName, pluginId)
}

func RemovePluginByAppId(appId string) error {
	return mongo.RemoveAll(pluginCollectionName, bson.M{"app_id": appId})
}

func NewPlugin(version string, content []byte, appId string) *Plugin {
	newMd5 := fmt.Sprintf("%x", md5.Sum(content))
	return &Plugin{Version: version, Md5: newMd5, Content: string(content)}
}
