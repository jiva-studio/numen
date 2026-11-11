package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/serving"
)

func updateConfig(model *serving.Server) func(values map[string]string, secrets server.Secrets) {
	return func(values map[string]string, secrets server.Secrets) {
		model.Update(values, getAPIKey(secrets))
	}
}

func getAPIKey(secrets server.Secrets) serving.Secret {
	list, isCarried := server.ReadSecrets(secrets, server.SecretAPIKey)
	if !isCarried {
		return serving.Secret{Value: "", IsSet: false}
	}
	if len(list) == 0 {
		return serving.Secret{Value: "", IsSet: true}
	}
	return serving.Secret{Value: list[0].Value, IsSet: true}
}
