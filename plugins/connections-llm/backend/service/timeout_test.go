package service

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestTheFirstRequestOfAnOpenAICompatibleModelOutlastsTheSettingAndTheNextOneDoesNot(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 300*time.Millisecond)
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.1"})
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")

	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err != nil {
		t.Fatalf("the first request: %v", err)
	}

	_, err := svc.Complete(newContext(t), call, oneUserMessage("hi"))
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestAFailedFirstRequestKeepsTheLongLimit(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 300*time.Millisecond)
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.1"})
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")
	fake.Fail("/chat/completions", 503, "")
	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err == nil {
		t.Fatal("want a failure")
	}
	fake.Fail("/chat/completions", 0, "")

	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err != nil {
		t.Errorf("the request after a failed first one: %v", err)
	}
}
