package rerank_test

import (
	"math"
	"slices"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/rerank"
)

func writeOldRequest(query, instruction string, documents []string) []byte {
	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	out = protowire.AppendString(out, query)
	if instruction != "" {
		out = protowire.AppendTag(out, 2, protowire.BytesType)
		out = protowire.AppendString(out, instruction)
	}
	for _, document := range documents {
		out = protowire.AppendTag(out, 3, protowire.BytesType)
		out = protowire.AppendString(out, document)
	}
	return out
}

func writeOldPackedResult(scores []float32) []byte {
	var packed []byte
	for _, score := range scores {
		packed = protowire.AppendFixed32(packed, math.Float32bits(score))
	}
	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, packed)
}

func readOldResult(t *testing.T, payload []byte) []float32 {
	t.Helper()
	var scores []float32
	for len(payload) > 0 {
		number, kind, n := protowire.ConsumeTag(payload)
		if n < 0 || number != 1 || kind != protowire.BytesType {
			t.Fatalf("the old reader cannot read the field %d of kind %d", number, kind)
		}
		packed, m := protowire.ConsumeBytes(payload[n:])
		payload = payload[n+m:]
		for len(packed) > 0 {
			bits, k := protowire.ConsumeFixed32(packed)
			packed = packed[k:]
			scores = append(scores, math.Float32frombits(bits))
		}
	}
	return scores
}

func TestTheRequestIsTheBytesOfTheOldLayout(t *testing.T) {
	documents := []string{"Ferry times", "Tide tables", ""}
	for _, instruction := range []string{"", "Find the notes that answer the question"} {
		got, err := EncodeRequest("when does the ferry leave", instruction, documents)
		if err != nil {
			t.Fatal(err)
		}

		if want := writeOldRequest("when does the ferry leave", instruction, documents); !slices.Equal(got, want) {
			t.Errorf("with instruction %q the request is % x, want the old layout % x", instruction, got, want)
		}
	}
}

func TestARequestOfTheOldLayoutDecodesWithTheMessage(t *testing.T) {
	payload := writeOldRequest("tide", "Find the notes", []string{"a", "b"})

	var got rerank.RerankRequest
	if err := proto.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}

	if got.GetQuery() != "tide" || got.GetInstruction() != "Find the notes" || !slices.Equal(got.GetDocuments(), []string{"a", "b"}) {
		t.Errorf("decoded %v, want the query, the instruction and both documents", &got)
	}
}

func TestAResultOfTheOldLayoutIsRead(t *testing.T) {
	scores := []float32{0.25, -1.5, 3}

	got, err := DecodeResult(writeOldPackedResult(scores))

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []float64{0.25, -1.5, 3}) {
		t.Errorf("scores %v, want %v", got, scores)
	}
}

func TestAResultOfTheMessageIsReadByTheOldLayout(t *testing.T) {
	payload, err := proto.Marshal(&rerank.RerankResult{Scores: []float32{0.5, 2}})
	if err != nil {
		t.Fatal(err)
	}

	if got := readOldResult(t, payload); !slices.Equal(got, []float32{0.5, 2}) {
		t.Errorf("the old reader got %v, want [0.5 2]", got)
	}
}

func TestAResultWithOneFieldForEachScoreIsRead(t *testing.T) {
	var payload []byte
	for _, score := range []float32{1, 2} {
		payload = protowire.AppendTag(payload, 1, protowire.Fixed32Type)
		payload = protowire.AppendFixed32(payload, math.Float32bits(score))
	}

	got, err := DecodeResult(payload)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []float64{1, 2}) {
		t.Errorf("scores %v, want [1 2]", got)
	}
}

func TestAResultThatIsNotAMessageIsAnError(t *testing.T) {
	for name, payload := range map[string][]byte{
		"a field cut short":       {0x0a, 0x08, 0x00},
		"a score cut short":       {0x0a, 0x03, 0x00, 0x00, 0x00},
		"a number that is no tag": {0x00},
	} {
		if _, err := DecodeResult(payload); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestATextThatIsNotUTF8CannotBeSent(t *testing.T) {
	if _, err := EncodeRequest("tide", "", []string{"a\xffb"}); err == nil {
		t.Error("no error for a document that is not UTF-8")
	}
}
