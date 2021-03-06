package server

import (
	"bytes"
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gin-gonic/gin"
	"github.com/orikami/config"
	"github.com/orikami/http"
	"github.com/orikami/secret"
	"github.com/orikami/storage"
	"github.com/spyzhov/ajson"
	"github.com/valyala/fasttemplate"
)

type Server struct {
	config            config.ServerConfig
	configRepoFactory storage.ConfigRepoFactory
}

func (svr Server) Run(cfg config.ServerConfig) {

	svr.config = cfg

	if cfg.Storage.Type == "etcd" {
		svr.configRepoFactory = storage.EtcdConfigRepoFactory
	}

	http.InitGin(func(e *gin.Engine) { initRoutes(e, svr) })
}

func (svr Server) Shutdown() {
}

func (svr Server) GetTemplate(store string, path string) string {
	storeOpt := svr.GetStoreOptions(store)
	templateStore := storage.GetTemplateStore(storeOpt)
	return templateStore.GetTemplate(path)
}

func (svr Server) PutTemplate(store string, path string, template string) string {

	storeOpt := svr.GetStoreOptions(store)
	templateStore := storage.GetTemplateStore(storeOpt)
	return templateStore.PutTemplate(path, template)
}

func (svr Server) DeleteTemplate(store string, path string) string {
	storeOpt := svr.GetStoreOptions(store)
	templateStore := storage.GetTemplateStore(storeOpt)
	return templateStore.DeleteTemplate(path)
}

func (svr Server) GetValues(store string, path string) string {
	storeOpt := svr.GetStoreOptions(store)
	valuesStore := storage.GetValuesStore(storeOpt)
	return valuesStore.GetValues(path)
}

func (svr Server) GetValuesInBatch(store string, paths []string) map[string]string {
	storeOpt := svr.GetStoreOptions(store)
	valuesStore := storage.GetValuesStore(storeOpt)
	return valuesStore.GetValuesInBatch(paths)
}

func (svr Server) PutValues(store string, path string, values string) string {
	storeOpt := svr.GetStoreOptions(store)
	valuesStore := storage.GetValuesStore(storeOpt)
	return valuesStore.PutValues(path, values)
}

func (svr Server) DeleteValues(store string, path string) string {
	storeOpt := svr.GetStoreOptions(store)
	valuesStore := storage.GetValuesStore(storeOpt)
	return valuesStore.DeleteValues(path)
}

func (svr Server) GetSecretsMap(engine string, store string, path string) string {
	if engine == "" {
		engine = "vault"
	}

	storeOpt := svr.GetStoreOptions(store)
	secretsMapStore := storage.GetSecretsMapStore(storeOpt)
	return secretsMapStore.GetSecretsMap(engine, path)
}

func (svr Server) PutSecretsMap(engine string, store string, path string, values string) string {
	if engine == "" {
		engine = "vault"
	}

	storeOpt := svr.GetStoreOptions(store)
	secretsMapStore := storage.GetSecretsMapStore(storeOpt)
	return secretsMapStore.PutSecretsMap(engine, path, values)
}

func (svr Server) DeleteSecretsMap(engine string, store string, path string) string {
	if engine == "" {
		engine = "vault"
	}

	storeOpt := svr.GetStoreOptions(store)
	secretsMapStore := storage.GetSecretsMapStore(storeOpt)
	return secretsMapStore.DeleteSecretsMap(engine, path)
}

func (svr Server) GetSecrets(engine string, store string, path string,
	userJsonOverrides string) string {
	if engine == "" {
		engine = "vault"
	}

	storeOpt := svr.GetStoreOptions(store)
	secretsMapStore := storage.GetSecretsMapStore(storeOpt)
	secretsMap := secretsMapStore.GetSecretsMap(engine, path)

	merged, _ := jsonpatch.MergePatch([]byte(secretsMap), []byte(userJsonOverrides))

	var opt map[string]interface{}
	json.Unmarshal(merged, &opt)

	secretOpt := svr.GetSecretsOptions(engine)
	secretClient := secret.GetSecretClient(secretOpt, opt)

	secret := secretClient.GetSecret(opt["path"].(string))

	jSecret, _ := json.Marshal(secret)
	root, _ := ajson.Unmarshal(jSecret)

	mapping := opt["map"].(map[string]interface{})
	result := make(map[string]interface{})

	for k, jsonPath := range mapping {

		nodes, _ := root.JSONPath(jsonPath.(string))
		size := len(nodes)

		if size == 1 {
			b, _ := ajson.Marshal(nodes[0])
			var intf interface{}
			json.Unmarshal(b, &intf)
			result[k] = intf
		} else if size > 1 {
			var objArray []interface{}
			for _, n := range nodes {
				b, _ := ajson.Marshal(n)
				var intf interface{}
				json.Unmarshal(b, &intf)
				objArray = append(objArray, intf)
			}
			result[k] = objArray
		}
	}

	r, _ := json.Marshal(result)

	return string(r)
}

func (svr Server) GetStoreOptions(store string) config.StoreOptions {

	var opt config.StoreOptions
	configRepo := svr.configRepoFactory(svr.config)

	if store == "" {
		optJson := configRepo.GetStoreOptions("default")
		if optJson != nil {
			json.Unmarshal(optJson, &opt)
		} else {
			return svr.config.Storage
		}
	} else {
		optJson := configRepo.GetStoreOptions(store)
		json.Unmarshal(optJson, &opt)
	}
	return opt
}

func (svr Server) PutStoreOptions(store string, optionsJson string) string {
	configRepo := svr.configRepoFactory(svr.config)
	if store == "" {
		store = "default"
	}
	return configRepo.PutStoreOptions(store, optionsJson)
}

func (svr Server) DeleteStoreOptions(store string) string {
	configRepo := svr.configRepoFactory(svr.config)
	if store == "" {
		store = "default"
	}
	return configRepo.DeleteStoreOptions(store)
}

func (svr Server) GetSecretsOptions(engine string) config.SecretsOptions {

	var opt config.SecretsOptions
	configRepo := svr.configRepoFactory(svr.config)

	if engine == "" {
		optJson := configRepo.GetSecretsOptions("default")
		if optJson != nil {
			json.Unmarshal(optJson, &opt)
		}

	} else {
		optJson := configRepo.GetSecretsOptions(engine)
		json.Unmarshal(optJson, &opt)
	}

	return opt
}

func (svr Server) PutSecretsOptions(engine string, optionsJson string) string {
	configRepo := svr.configRepoFactory(svr.config)
	if engine == "" {
		engine = "default"
	}
	return configRepo.PutSecretsOptions(engine, optionsJson)
}

func (svr Server) DeleteSecretsOptions(engine string) string {
	configRepo := svr.configRepoFactory(svr.config)
	if engine == "" {
		engine = "default"
	}
	return configRepo.DeleteSecretsOptions(engine)
}

func (svr Server) Render(store string, templatePath string, render RenderDto) (result string, ok bool) {

	templateValueMap := make(map[string]interface{})
	storeValueMap := prefetchValuesByBatch(render, svr)

	rawTmpl := svr.GetTemplate(store, templatePath)

	values := render.Values
	for valueIndex := len(values) - 1; valueIndex >= 0; valueIndex-- {
		val := values[valueIndex]
		storeKeys := val.StoreKeys

		if len(storeKeys) == 0 {
			storeKeys = append(storeKeys, store)
		}

		for storeIndex := len(storeKeys) - 1; storeIndex >= 0; storeIndex-- {
			store := storeKeys[storeIndex]

			rawValue := storeValueMap[store][val.Path]
			var vm map[string]interface{}
			json.Unmarshal([]byte(rawValue), &vm)
			for k, v := range vm {
				templateValueMap[k] = v
			}
		}

		// parse and get values and secrets
		// need to build a tree and prevent recursions
	}

	tmpl := fasttemplate.New(rawTmpl, "{{", "}}")

	buf := new(bytes.Buffer)
	tmpl.Execute(buf, templateValueMap)
	return buf.String(), true
}

func prefetchValuesByBatch(render RenderDto, svr Server) map[string]map[string]string {

	//storeMap[store] = valuePath
	storeMap := make(map[string][]string)
	for _, val := range render.Values {
		storeKeys := val.StoreKeys
		if len(storeKeys) == 0 {
			storeKeys = append(storeKeys, "")
		}
		for _, s := range storeKeys {
			storeMap[s] = append(storeMap[s], val.Path)
		}
	}

	//storeValueMap[store][valuePath] = Value
	storeValueMap := make(map[string]map[string]string)
	for store, valuePaths := range storeMap {
		valMap := svr.GetValuesInBatch(store, valuePaths)

		for valuePath, val := range valMap {
			if storeValueMap[store] == nil {
				storeValueMap[store] = make(map[string]string)
			}
			storeValueMap[store][valuePath] = val
		}
	}

	return storeValueMap
}
